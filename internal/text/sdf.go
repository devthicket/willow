package text

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// ---------------------------------------------------------------------------
// SDF generation types
// ---------------------------------------------------------------------------

// SDFGenOptions configures SDF atlas generation.
type SDFGenOptions struct {
	// Size is the font size in pixels for rasterization.
	Size float64
	// DistanceRange is the pixel range of the distance field (typically 4-8).
	DistanceRange float64
	// Chars is the set of characters to include. Empty means printable ASCII (32-126).
	Chars string
	// AtlasWidth is the maximum atlas width in pixels. Default 1024.
	AtlasWidth int
	// PageIndex is the atlas page index for the generated font. Default 0.
	PageIndex uint16
	// MSDF enables multi-channel SDF generation (not yet implemented).
	MSDF bool
}

// Defaults fills in zero-valued options with sensible defaults.
func (o *SDFGenOptions) Defaults() {
	if o.Size <= 0 {
		o.Size = 80
	}
	if o.DistanceRange <= 0 {
		o.DistanceRange = 8
	}
	if o.AtlasWidth <= 0 {
		o.AtlasWidth = 1024
	}
	if o.Chars == "" {
		var buf []byte
		for c := byte(32); c <= 126; c++ {
			buf = append(buf, c)
		}
		o.Chars = string(buf)
	}
}

// GlyphBitmap holds a rasterized glyph and its metrics for atlas packing.
type GlyphBitmap struct {
	ID       rune
	Img      *image.Alpha // rasterized glyph (grayscale)
	XOffset  int
	YOffset  int
	XAdvance int
	// Atlas position (filled during packing)
	atlasX, atlasY int
}

// ---------------------------------------------------------------------------
// SDF generation algorithm
// ---------------------------------------------------------------------------

// GenerateSDFFromBitmaps creates an SDF atlas from pre-rasterized glyph bitmaps.
// This is the core algorithm used by both runtime and offline generation paths.
// Each GlyphBitmap should have its Img field set to the rasterized glyph.
//
// Returns the atlas image and metrics JSON.
func GenerateSDFFromBitmaps(glyphs []GlyphBitmap, opts SDFGenOptions) (*image.NRGBA, []byte, error) {
	opts.Defaults()
	pad := int(opts.DistanceRange) + 1

	// Compute padded glyph sizes for atlas packing
	type packedGlyph struct {
		*GlyphBitmap
		pw, ph int // padded width, height
	}
	packed := make([]packedGlyph, len(glyphs))
	for i := range glyphs {
		g := &glyphs[i]
		w, h := 0, 0
		if g.Img != nil {
			b := g.Img.Bounds()
			w, h = b.Dx(), b.Dy()
		}
		packed[i] = packedGlyph{
			GlyphBitmap: g,
			pw:          w + 2*pad,
			ph:          h + 2*pad,
		}
	}

	// Sort by height descending for shelf packing
	sort.Slice(packed, func(i, j int) bool {
		return packed[i].ph > packed[j].ph
	})

	// Shelf-pack into atlas
	atlasW := opts.AtlasWidth
	shelfY := 0
	shelfH := 0
	cursorX := 0

	for i := range packed {
		g := &packed[i]
		if cursorX+g.pw > atlasW {
			// New shelf
			shelfY += shelfH
			shelfH = 0
			cursorX = 0
		}
		g.atlasX = cursorX
		g.atlasY = shelfY
		cursorX += g.pw
		if g.ph > shelfH {
			shelfH = g.ph
		}
	}
	atlasH := shelfY + shelfH
	if atlasH == 0 {
		atlasH = 1
	}

	// Create atlas image
	atlas := image.NewNRGBA(image.Rect(0, 0, atlasW, atlasH))

	// Build metrics
	type jsonGlyph struct {
		ID       int `json:"id"`
		X        int `json:"x"`
		Y        int `json:"y"`
		Width    int `json:"width"`
		Height   int `json:"height"`
		XOffset  int `json:"xOffset"`
		YOffset  int `json:"yOffset"`
		XAdvance int `json:"xAdvance"`
	}

	jsonGlyphs := make([]jsonGlyph, 0, len(packed))

	for i := range packed {
		g := &packed[i]

		// Compute SDF for this glyph and write to atlas
		if g.Img != nil {
			sdf := computeEDT(g.Img, opts.DistanceRange)
			// Write SDF to atlas at the glyph's packed position
			for y := 0; y < g.ph; y++ {
				for x := 0; x < g.pw; x++ {
					ax := g.atlasX + x
					ay := g.atlasY + y
					if ax < atlasW && ay < atlasH {
						d := sdf[y*g.pw+x]
						// Encode as alpha (white pixel, distance in alpha)
						atlas.SetNRGBA(ax, ay, color.NRGBA{R: 255, G: 255, B: 255, A: d})
					}
				}
			}
		}

		// Record the padded region in the atlas
		jsonGlyphs = append(jsonGlyphs, jsonGlyph{
			ID:       int(g.ID),
			X:        g.atlasX,
			Y:        g.atlasY,
			Width:    g.pw,
			Height:   g.ph,
			XOffset:  g.XOffset - pad,
			YOffset:  g.YOffset - pad,
			XAdvance: g.XAdvance,
		})
	}

	// Compute line metrics from glyph data
	lineHeight := opts.Size * 1.2 // reasonable default
	base := opts.Size

	sdfType := "sdf"
	if opts.MSDF {
		sdfType = "msdf"
	}

	metrics := sdfMetrics{
		Type:          sdfType,
		Size:          opts.Size,
		DistanceRange: opts.DistanceRange,
		LineHeight:    lineHeight,
		Base:          base,
		Glyphs:        make([]sdfGlyphEntry, len(jsonGlyphs)),
	}
	for i, jg := range jsonGlyphs {
		metrics.Glyphs[i] = sdfGlyphEntry{
			ID:       jg.ID,
			X:        jg.X,
			Y:        jg.Y,
			Width:    jg.Width,
			Height:   jg.Height,
			XOffset:  jg.XOffset,
			YOffset:  jg.YOffset,
			XAdvance: jg.XAdvance,
		}
	}

	metricsJSON, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	return atlas, metricsJSON, nil
}

// computeEDT computes a signed distance field from an alpha bitmap using the
// dead-reckoning Euclidean Distance Transform algorithm.
// Returns a flat slice of uint8 values representing the distance field
// for a padded region (pad = distanceRange + 1 on each side).
func computeEDT(src *image.Alpha, distanceRange float64) []uint8 {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	pad := int(distanceRange) + 1
	w := srcW + 2*pad
	h := srcH + 2*pad

	// Initialize distance grids for inside and outside
	inf := float64(w + h)
	outside := make([]float64, w*h) // distance to nearest edge from outside
	inside := make([]float64, w*h)  // distance to nearest edge from inside

	for i := range outside {
		outside[i] = inf
		inside[i] = inf
	}

	// Determine inside/outside from source alpha
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := x - pad
			sy := y - pad
			isInside := false
			if sx >= 0 && sx < srcW && sy >= 0 && sy < srcH {
				a := src.AlphaAt(bounds.Min.X+sx, bounds.Min.Y+sy).A
				isInside = a > 127
			}

			idx := y*w + x
			if isInside {
				outside[idx] = 0 // on the shape = 0 outside distance
				// inside[idx] stays inf (will be computed)
			} else {
				inside[idx] = 0 // outside the shape = 0 inside distance
				// outside[idx] stays inf (will be computed)
			}
		}
	}

	// Apply 2D EDT to both grids
	edt2d(outside, w, h)
	edt2d(inside, w, h)

	// Combine into signed distance and normalize to [0, 255]
	result := make([]uint8, w*h)
	for i := range result {
		d := math.Sqrt(inside[i]) - math.Sqrt(outside[i])
		// Map [-distanceRange, +distanceRange] to [0, 255] with 0.5 at edge
		normalized := 0.5 + 0.5*(d/distanceRange)
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}
		result[i] = uint8(normalized*255 + 0.5)
	}

	return result
}

// edt2d computes the squared Euclidean Distance Transform in-place using
// Felzenszwalb & Huttenlocher's O(n) algorithm (separable 1D passes).
func edt2d(grid []float64, w, h int) {
	// Temporary buffers for 1D EDT
	maxDim := w
	if h > maxDim {
		maxDim = h
	}
	f := make([]float64, maxDim)
	z := make([]float64, maxDim+1)
	v := make([]int, maxDim)

	// Transform columns
	col := make([]float64, h)
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			col[y] = grid[y*w+x]
		}
		edt1d(col, f, z, v, h)
		for y := 0; y < h; y++ {
			grid[y*w+x] = col[y]
		}
	}

	// Transform rows
	row := make([]float64, w)
	for y := 0; y < h; y++ {
		copy(row, grid[y*w:(y+1)*w])
		edt1d(row, f, z, v, w)
		copy(grid[y*w:], row)
	}
}

// edt1d computes 1D squared distance transform using the parabola envelope method.
// data is the input/output array, f/z/v are temporary buffers.
func edt1d(data []float64, f, z []float64, v []int, n int) {
	if n == 0 {
		return
	}

	// Build lower envelope of parabolas
	for i := 0; i < n; i++ {
		f[i] = data[i]
	}

	v[0] = 0
	z[0] = -1e20
	z[1] = 1e20
	k := 0

	for q := 1; q < n; q++ {
		for {
			s := ((f[q] + float64(q*q)) - (f[v[k]] + float64(v[k]*v[k]))) / (2*float64(q) - 2*float64(v[k]))
			if s > z[k] {
				k++
				v[k] = q
				z[k] = s
				z[k+1] = 1e20
				break
			}
			k--
		}
	}

	// Fill in values from the lower envelope
	k = 0
	for q := 0; q < n; q++ {
		for z[k+1] < float64(q) {
			k++
		}
		dx := float64(q - v[k])
		data[q] = dx*dx + f[v[k]]
	}
}

// fixedToFloat64 converts a fixed.Int26_6 value to float64.
func fixedToFloat64(f fixed.Int26_6) float64 {
	return float64(f) / 64.0
}

// ---------------------------------------------------------------------------
// LoadSpriteFontFromTTF
// ---------------------------------------------------------------------------

// LoadSpriteFontFromTTF generates an SDF font atlas at runtime from TTF/OTF data.
// Uses pure-Go rasterization (golang.org/x/image/font/opentype) so it can be
// called before the Ebitengine game loop starts. Returns the SpriteFont, the atlas
// as an ebiten.Image (caller must register via Scene.RegisterPage), the raw
// atlas as an *image.NRGBA (for saving to disk), and any error.
func LoadSpriteFontFromTTF(ttfData []byte, opts SDFGenOptions) (*SpriteFont, *ebiten.Image, *image.NRGBA, error) {
	opts.Defaults()

	// Parse font with golang.org/x/image/font/opentype (pure Go, no GPU needed).
	otFont, err := opentype.Parse(ttfData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("text: failed to parse TTF data: %w", err)
	}

	face, err := opentype.NewFace(otFont, &opentype.FaceOptions{
		Size:    opts.Size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("text: failed to create font face: %w", err)
	}
	defer face.Close()

	faceMetrics := face.Metrics()
	ascent := fixedToFloat64(faceMetrics.Ascent)
	descent := fixedToFloat64(faceMetrics.Descent)
	lineHeight := fixedToFloat64(faceMetrics.Height)
	if lineHeight <= 0 {
		lineHeight = ascent + descent
	}

	// Rasterize each glyph using pure-Go font.Drawer → image.RGBA.
	var glyphs []GlyphBitmap
	for i := 0; i < len(opts.Chars); {
		r, sz := utf8.DecodeRuneInString(opts.Chars[i:])
		i += sz

		bounds, advance, ok := face.GlyphBounds(r)
		if !ok {
			continue
		}

		x0 := bounds.Min.X.Floor()
		y0 := bounds.Min.Y.Floor()
		x1 := bounds.Max.X.Ceil()
		y1 := bounds.Max.Y.Ceil()
		gw := x1 - x0
		gh := y1 - y0

		if gw <= 0 || gh <= 0 {
			// Whitespace or empty glyph — record advance only.
			glyphs = append(glyphs, GlyphBitmap{
				ID:       r,
				Img:      nil,
				XOffset:  0,
				YOffset:  0,
				XAdvance: advance.Ceil(),
			})
			continue
		}

		// Rasterize to RGBA using font.Drawer (CPU only, no Ebitengine).
		rgba := image.NewRGBA(image.Rect(0, 0, gw, gh))
		d := font.Drawer{
			Dst:  rgba,
			Src:  image.White,
			Face: face,
			Dot:  fixed.Point26_6{X: -bounds.Min.X, Y: -bounds.Min.Y},
		}
		d.DrawString(string(r))

		// Extract alpha channel directly from RGBA pixel data.
		alphaImg := image.NewAlpha(image.Rect(0, 0, gw, gh))
		for py := 0; py < gh; py++ {
			for px := 0; px < gw; px++ {
				off := rgba.PixOffset(px, py)
				alphaImg.Pix[py*alphaImg.Stride+px] = rgba.Pix[off+3]
			}
		}

		glyphs = append(glyphs, GlyphBitmap{
			ID:       r,
			Img:      alphaImg,
			XOffset:  x0,
			YOffset:  y0 + faceMetrics.Ascent.Ceil(),
			XAdvance: advance.Ceil(),
		})
	}

	atlas, metricsJSON, err := GenerateSDFFromBitmaps(glyphs, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("text: SDF generation failed: %w", err)
	}

	// Override lineHeight and base with actual font metrics.
	var m sdfMetrics
	_ = json.Unmarshal(metricsJSON, &m)
	m.LineHeight = lineHeight
	m.Base = ascent
	metricsJSON, _ = json.Marshal(m)

	sdfFont, err := LoadSpriteFont(metricsJSON, opts.PageIndex)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("text: failed to load generated SDF font: %w", err)
	}

	// Convert image.NRGBA → ebiten.Image (WritePixels works before game start).
	atlasImg := ebiten.NewImage(atlas.Bounds().Dx(), atlas.Bounds().Dy())
	atlasImg.WritePixels(atlas.Pix)

	return sdfFont, atlasImg, atlas, nil
}
