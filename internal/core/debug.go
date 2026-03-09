package core

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/render"
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

func writePNG(path string, img *image.NRGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
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
