package core

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/draw"
)

// GifConfig controls GIF recording behaviour.
type GifConfig struct {
	FrameSkip int // capture every Nth frame; 0 defaults to 2
	MaxWidth  int // downscale target width; 0 defaults to 480
	MaxHeight int // downscale target height; 0 defaults to 360
}

// GifRecorder accumulates frames from the ebiten screen and writes an
// optimised animated GIF on Finish.
type GifRecorder struct {
	label  string
	dir    string
	config GifConfig

	tick      int // frame counter for skip logic
	frames    []*image.NRGBA
	delays    []int // centiseconds per accumulated frame
	pendingCS int   // accumulated delay for skipped/deduped frames
	scaledW   int
	scaledH   int

	// Reusable buffers.
	pixBuf []byte
	fullBuf *image.NRGBA // full-resolution capture buffer
}

// NewGifRecorder creates a recorder.  Frames are captured every config.FrameSkip
// ticks and downscaled to fit within MaxWidth × MaxHeight.
func NewGifRecorder(label, dir string, cfg GifConfig) *GifRecorder {
	if cfg.FrameSkip <= 0 {
		cfg.FrameSkip = 2
	}
	if cfg.MaxWidth <= 0 {
		cfg.MaxWidth = 480
	}
	if cfg.MaxHeight <= 0 {
		cfg.MaxHeight = 360
	}
	return &GifRecorder{label: label, dir: dir, config: cfg}
}

// CaptureFrame is called every Draw().  It decides whether to sample this
// frame based on FrameSkip, reads pixels, downscales, and stores the result.
func (g *GifRecorder) CaptureFrame(screen *ebiten.Image) {
	csPerTick := 100 / int(ebiten.TPS()) // centiseconds per engine tick
	if csPerTick < 1 {
		csPerTick = 1
	}
	g.pendingCS += csPerTick

	g.tick++
	if g.tick%g.config.FrameSkip != 0 {
		return
	}

	bounds := screen.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	// Compute scaled size on first capture.
	if g.scaledW == 0 {
		g.scaledW, g.scaledH = fitDimensions(srcW, srcH, g.config.MaxWidth, g.config.MaxHeight)
	}

	// Read pixels from the ebiten screen.
	need := 4 * srcW * srcH
	if len(g.pixBuf) < need {
		g.pixBuf = make([]byte, need)
	}
	screen.ReadPixels(g.pixBuf)

	// De-premultiply into a reusable NRGBA buffer.
	if g.fullBuf == nil || g.fullBuf.Rect.Dx() != srcW || g.fullBuf.Rect.Dy() != srcH {
		g.fullBuf = image.NewNRGBA(image.Rect(0, 0, srcW, srcH))
	}
	depremultiply(g.pixBuf, g.fullBuf.Pix)

	// Downscale.
	scaled := image.NewNRGBA(image.Rect(0, 0, g.scaledW, g.scaledH))
	draw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), g.fullBuf, g.fullBuf.Bounds(), draw.Over, nil)

	// Dedup: if this frame is nearly identical to the previous one, merge delay.
	if len(g.frames) > 0 {
		prev := g.frames[len(g.frames)-1]
		if frameSimilar(prev, scaled, 0.01) {
			g.delays[len(g.delays)-1] += g.pendingCS
			g.pendingCS = 0
			return
		}
	}

	g.frames = append(g.frames, scaled)
	g.delays = append(g.delays, g.pendingCS)
	g.pendingCS = 0
}

// Finish encodes the accumulated frames as an optimised GIF and writes it.
func (g *GifRecorder) Finish() error {
	if len(g.frames) == 0 {
		return nil
	}

	if err := os.MkdirAll(g.dir, 0o755); err != nil {
		return fmt.Errorf("gif mkdir: %w", err)
	}

	// Build a global palette from all frames.
	palette := buildGlobalPalette(g.frames, 255)
	// Index 0 is reserved for transparent.
	palette[0] = color.NRGBA{0, 0, 0, 0}

	anim := gif.GIF{LoopCount: 0}

	var prevPaletted *image.Paletted

	for i, frame := range g.frames {
		paletted := quantizeFrame(frame, palette)

		if i == 0 {
			// First frame: full image.
			anim.Image = append(anim.Image, paletted)
			anim.Delay = append(anim.Delay, g.delays[i])
			anim.Disposal = append(anim.Disposal, gif.DisposalNone)
		} else {
			// Diff frame: only encode changed pixels.
			diff, rect := diffFrame(prevPaletted, paletted, 0)
			sub := image.NewPaletted(rect, palette)
			for y := rect.Min.Y; y < rect.Max.Y; y++ {
				for x := rect.Min.X; x < rect.Max.X; x++ {
					sub.SetColorIndex(x, y, diff.ColorIndexAt(x, y))
				}
			}
			anim.Image = append(anim.Image, sub)
			anim.Delay = append(anim.Delay, g.delays[i])
			anim.Disposal = append(anim.Disposal, gif.DisposalNone)
		}

		prevPaletted = paletted
	}

	stamp := time.Now().Format("20060102_150405")
	safe := SanitizeLabel(g.label)
	path := fmt.Sprintf("%s/%s_%s.gif", g.dir, stamp, safe)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("gif create: %w", err)
	}
	if err := gif.EncodeAll(f, &anim); err != nil {
		_ = f.Close()
		return fmt.Errorf("gif encode: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("gif close: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stderr, "[willow] gif: %d frames → %s\n", len(anim.Image), path)

	// Release memory.
	g.frames = nil
	g.delays = nil
	g.pixBuf = nil
	g.fullBuf = nil
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// fitDimensions returns dimensions that fit within maxW×maxH while preserving
// aspect ratio.  If src already fits, returns src dimensions.
func fitDimensions(srcW, srcH, maxW, maxH int) (int, int) {
	if srcW <= maxW && srcH <= maxH {
		return srcW, srcH
	}
	ratioW := float64(maxW) / float64(srcW)
	ratioH := float64(maxH) / float64(srcH)
	ratio := ratioW
	if ratioH < ratio {
		ratio = ratioH
	}
	w := int(float64(srcW) * ratio)
	h := int(float64(srcH) * ratio)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// depremultiply converts premultiplied-alpha RGBA bytes to straight NRGBA.
func depremultiply(src, dst []byte) {
	for i := 0; i < len(src); i += 4 {
		r, g, b, a := src[i], src[i+1], src[i+2], src[i+3]
		if a > 0 && a < 255 {
			r = uint8(min(int(r)*255/int(a), 255))
			g = uint8(min(int(g)*255/int(a), 255))
			b = uint8(min(int(b)*255/int(a), 255))
		}
		dst[i] = r
		dst[i+1] = g
		dst[i+2] = b
		dst[i+3] = a
	}
}

// frameSimilar returns true if the fraction of differing pixels is below threshold.
func frameSimilar(a, b *image.NRGBA, threshold float64) bool {
	if a.Rect != b.Rect {
		return false
	}
	total := len(a.Pix) / 4
	diff := 0
	// Sample every 4th pixel for speed.
	for i := 0; i < len(a.Pix); i += 16 {
		if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] ||
			a.Pix[i+2] != b.Pix[i+2] {
			diff++
		}
	}
	sampled := total / 4
	if sampled == 0 {
		return true
	}
	return float64(diff)/float64(sampled) < threshold
}

// ---------------------------------------------------------------------------
// Palette generation (median-cut)
// ---------------------------------------------------------------------------

type colorBox struct {
	colors [][3]uint8
}

func (b *colorBox) longestAxis() int {
	var minR, minG, minB uint8 = 255, 255, 255
	var maxR, maxG, maxB uint8
	for _, c := range b.colors {
		if c[0] < minR {
			minR = c[0]
		}
		if c[0] > maxR {
			maxR = c[0]
		}
		if c[1] < minG {
			minG = c[1]
		}
		if c[1] > maxG {
			maxG = c[1]
		}
		if c[2] < minB {
			minB = c[2]
		}
		if c[2] > maxB {
			maxB = c[2]
		}
	}
	dr := int(maxR) - int(minR)
	dg := int(maxG) - int(minG)
	db := int(maxB) - int(minB)
	if dr >= dg && dr >= db {
		return 0
	}
	if dg >= db {
		return 1
	}
	return 2
}

func (b *colorBox) average() color.NRGBA {
	var r, g, bl int
	n := len(b.colors)
	if n == 0 {
		return color.NRGBA{}
	}
	for _, c := range b.colors {
		r += int(c[0])
		g += int(c[1])
		bl += int(c[2])
	}
	return color.NRGBA{uint8(r / n), uint8(g / n), uint8(bl / n), 255}
}

// buildGlobalPalette samples all frames and produces a median-cut palette.
func buildGlobalPalette(frames []*image.NRGBA, maxColors int) color.Palette {
	// Sample pixels from all frames.
	seen := make(map[[3]uint8]struct{})
	for _, frame := range frames {
		step := 4 // sample every 4th pixel
		if len(frame.Pix)/4 > 100000 {
			step = 8
		}
		for i := 0; i < len(frame.Pix); i += step * 4 {
			r, g, b := frame.Pix[i], frame.Pix[i+1], frame.Pix[i+2]
			// Quantize to 6-bit to reduce unique count.
			key := [3]uint8{r & 0xFC, g & 0xFC, b & 0xFC}
			seen[key] = struct{}{}
		}
	}

	colors := make([][3]uint8, 0, len(seen))
	for c := range seen {
		colors = append(colors, c)
	}

	if len(colors) <= maxColors {
		pal := make(color.Palette, len(colors)+1)
		pal[0] = color.NRGBA{} // transparent at index 0
		for i, c := range colors {
			pal[i+1] = color.NRGBA{c[0], c[1], c[2], 255}
		}
		return pal
	}

	// Median-cut.
	boxes := []colorBox{{colors: colors}}
	for len(boxes) < maxColors {
		// Find the largest box.
		bestIdx := 0
		bestLen := 0
		for i, bx := range boxes {
			if len(bx.colors) > bestLen {
				bestLen = len(bx.colors)
				bestIdx = i
			}
		}
		if bestLen < 2 {
			break
		}

		bx := boxes[bestIdx]
		axis := bx.longestAxis()
		sort.Slice(bx.colors, func(i, j int) bool {
			return bx.colors[i][axis] < bx.colors[j][axis]
		})
		mid := len(bx.colors) / 2
		boxes[bestIdx] = colorBox{colors: bx.colors[:mid]}
		boxes = append(boxes, colorBox{colors: bx.colors[mid:]})
	}

	pal := make(color.Palette, len(boxes)+1)
	pal[0] = color.NRGBA{} // transparent
	for i, bx := range boxes {
		pal[i+1] = bx.average()
	}
	return pal
}

// quantizeFrame maps an NRGBA image to a paletted image using nearest-colour.
func quantizeFrame(src *image.NRGBA, pal color.Palette) *image.Paletted {
	dst := image.NewPaletted(src.Rect, pal)
	// Build a small cache to avoid repeated palette lookups.
	cache := make(map[[3]uint8]uint8, 256)
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			off := src.PixOffset(x, y)
			r, g, b := src.Pix[off], src.Pix[off+1], src.Pix[off+2]
			key := [3]uint8{r & 0xFC, g & 0xFC, b & 0xFC}
			idx, ok := cache[key]
			if !ok {
				idx = uint8(pal.Index(color.NRGBA{r, g, b, 255}))
				// Never map to index 0 (transparent) for opaque pixels.
				if idx == 0 && src.Pix[off+3] > 0 {
					idx = 1
				}
				cache[key] = idx
			}
			dst.Pix[dst.PixOffset(x, y)] = idx
		}
	}
	return dst
}

// diffFrame produces a paletted image where unchanged pixels are set to
// transparentIdx.  Returns the diff image and the bounding rect of changes.
func diffFrame(prev, cur *image.Paletted, transparentIdx uint8) (*image.Paletted, image.Rectangle) {
	rect := cur.Rect
	diff := image.NewPaletted(rect, cur.Palette)

	minX, minY := rect.Max.X, rect.Max.Y
	maxX, maxY := rect.Min.X, rect.Min.Y
	hasChange := false

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			ci := cur.ColorIndexAt(x, y)
			pi := prev.ColorIndexAt(x, y)
			if ci == pi {
				diff.SetColorIndex(x, y, transparentIdx)
			} else {
				diff.SetColorIndex(x, y, ci)
				hasChange = true
				if x < minX {
					minX = x
				}
				if x+1 > maxX {
					maxX = x + 1
				}
				if y < minY {
					minY = y
				}
				if y+1 > maxY {
					maxY = y + 1
				}
			}
		}
	}

	if !hasChange {
		// Entirely identical — return a 1×1 transparent frame.
		tiny := image.NewPaletted(image.Rect(0, 0, 1, 1), cur.Palette)
		tiny.Pix[0] = transparentIdx
		return tiny, tiny.Rect
	}

	return diff, image.Rect(minX, minY, maxX, maxY)
}
