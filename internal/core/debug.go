package core

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/devthicket/willow/internal/render"
	"github.com/hajimehoshi/ebiten/v2"
)

// DebugStats holds per-frame timing and draw-call metrics.
type DebugStats struct {
	TraverseTime  time.Duration
	SortTime      time.Duration
	BatchTime     time.Duration
	SubmitTime    time.Duration
	CommandCount  int
	BatchCount    int
	DrawCallCount int
}

// DebugLog prints timing and draw-call stats to stderr.
func DebugLog(stats DebugStats) {
	total := stats.TraverseTime + stats.SortTime + stats.BatchTime + stats.SubmitTime
	_, _ = fmt.Fprintf(os.Stderr,
		"[willow] traverse: %v | sort: %v | batch: %v | submit: %v | total: %v\n",
		stats.TraverseTime, stats.SortTime, stats.BatchTime, stats.SubmitTime, total)
	_, _ = fmt.Fprintf(os.Stderr,
		"[willow] commands: %d | batches: %d | draw calls: %d\n",
		stats.CommandCount, stats.BatchCount, stats.DrawCallCount)
}

// CountBatches counts contiguous groups of commands sharing the same BatchKey.
func CountBatches(commands []render.RenderCommand) int {
	if len(commands) == 0 {
		return 0
	}
	count := 1
	prev := render.CommandBatchKey(&commands[0])
	for i := 1; i < len(commands); i++ {
		cur := render.CommandBatchKey(&commands[i])
		if cur != prev {
			count++
			prev = cur
		}
	}
	return count
}

// CountDrawCalls counts individual draw calls from the command list.
func CountDrawCalls(commands []render.RenderCommand) int {
	count := 0
	for i := range commands {
		cmd := &commands[i]
		switch cmd.Type {
		case render.CommandParticle:
			if cmd.Emitter != nil {
				count += cmd.Emitter.Alive
			}
		default:
			count++
		}
	}
	return count
}

// CountDrawCallsCoalesced estimates actual draw calls in coalesced mode.
func CountDrawCallsCoalesced(commands []render.RenderCommand) int {
	if len(commands) == 0 {
		return 0
	}
	count := 0
	inSpriteRun := false
	var prevKey render.BatchKey
	for i := range commands {
		cmd := &commands[i]
		switch cmd.Type {
		case render.CommandSprite:
			if cmd.DirectImage != nil {
				if inSpriteRun {
					count++
					inSpriteRun = false
				}
				count++
			} else {
				key := render.CommandBatchKey(cmd)
				if !inSpriteRun || key != prevKey {
					if inSpriteRun {
						count++
					}
					inSpriteRun = true
					prevKey = key
				}
			}
		case render.CommandParticle:
			if inSpriteRun {
				count++
				inSpriteRun = false
			}
			if cmd.Emitter != nil && cmd.Emitter.Alive > 0 {
				count++
			}
		case render.CommandMesh:
			if inSpriteRun {
				count++
				inSpriteRun = false
			}
			count++
		case render.CommandSDF:
			if inSpriteRun {
				count++
				inSpriteRun = false
			}
			count++
		case render.CommandBitmapText:
			if inSpriteRun {
				count++
				inSpriteRun = false
			}
			count++
		}
	}
	if inSpriteRun {
		count++
	}
	return count
}

// --- Screenshot ---

// FlushScreenshots captures the rendered frame for every queued label and
// writes each as a PNG file.
func FlushScreenshots(screen *ebiten.Image, queue []string, dir string) {
	if len(queue) == 0 {
		return
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[willow] screenshot: mkdir %s: %v\n", dir, err)
		return
	}

	bounds := screen.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	pixels := make([]byte, 4*w*h)
	screen.ReadPixels(pixels)

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pixels); i += 4 {
		r, g, b, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
		if a > 0 && a < 255 {
			r = uint8(min(int(r)*255/int(a), 255))
			g = uint8(min(int(g)*255/int(a), 255))
			b = uint8(min(int(b)*255/int(a), 255))
		}
		img.Pix[i] = r
		img.Pix[i+1] = g
		img.Pix[i+2] = b
		img.Pix[i+3] = a
	}

	stamp := time.Now().Format("20060102_150405")

	for _, label := range queue {
		safe := SanitizeLabel(label)
		path := fmt.Sprintf("%s/%s_%s.png", dir, stamp, safe)
		if err := writePNG(path, img); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "[willow] screenshot: %v\n", err)
		}
	}
}

// FlushPixelChecks reads pixels from the rendered screen and checks each
// pending assertion. Colors are matched loosely (dominant channel).
func FlushPixelChecks(screen *ebiten.Image, checks []PendingPixelCheck) {
	if len(checks) == 0 {
		return
	}
	bounds := screen.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	pixels := make([]byte, 4*w*h)
	screen.ReadPixels(pixels)

	for _, pc := range checks {
		if pc.X < 0 || pc.X >= w || pc.Y < 0 || pc.Y >= h {
			fmt.Fprintf(os.Stderr, "[pixelcheck] FAIL (%d,%d): out of bounds (%dx%d)\n", pc.X, pc.Y, w, h)
			continue
		}
		off := (pc.Y*w + pc.X) * 4
		r, g, b := pixels[off], pixels[off+1], pixels[off+2]
		actual := classifyColor(r, g, b)
		expected := pc.Color
		if matchColor(actual, r, g, b, expected) {
			fmt.Fprintf(os.Stderr, "[pixelcheck] OK   (%d,%d): expected=%s actual=%s rgb(%d,%d,%d)\n", pc.X, pc.Y, expected, actual, r, g, b)
		} else {
			fmt.Fprintf(os.Stderr, "[pixelcheck] FAIL (%d,%d): expected=%s actual=%s rgb(%d,%d,%d)\n", pc.X, pc.Y, expected, actual, r, g, b)
		}
	}
}

// classifyColor returns a human-readable name for the dominant color channel.
func classifyColor(r, g, b uint8) string {
	if r > 150 && r > g+20 && r > b+20 {
		return "red"
	}
	if g > 150 && r < 100 && b < 100 {
		return "green"
	}
	if b > 150 && r < 100 && g < 100 {
		return "blue"
	}
	if r > 200 && g > 200 && b > 200 {
		return "white"
	}
	if r < 50 && g < 50 && b < 50 {
		return "black"
	}
	if r > 150 && g > 150 && b < 100 {
		return "yellow"
	}
	avg := (int(r) + int(g) + int(b)) / 3
	diff := abs(int(r)-avg) + abs(int(g)-avg) + abs(int(b)-avg)
	if diff < 30 {
		return "grey"
	}
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}

func matchColor(actual string, r, g, b uint8, expected string) bool {
	// Exact match on classified name
	if actual == expected {
		return true
	}
	// "reddish", "bluish", etc. — check dominant channel loosely
	switch expected {
	case "reddish":
		return r > g+30 && r > b+30
	case "bluish":
		return b > r+30 && b > g+30
	case "greenish":
		return g > r+30 && g > b+30
	case "dark":
		return r < 80 && g < 80 && b < 80
	case "light":
		return r > 170 && g > 170 && b > 170
	}
	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func writePNG(path string, img *image.NRGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return f.Close()
}

// SanitizeLabel replaces characters that are unsafe in file names with
// underscores and falls back to "unlabeled" for empty strings.
func SanitizeLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "unlabeled"
	}
	var b strings.Builder
	b.Grow(len(label))
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9', r == '-', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
