package willow

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"image/color"
)

// NewFPSWidget creates a new Node that displays the current FPS and TPS.
// The widget is transparently updated every ~0.5 seconds.
// It uses a custom internal image and ebitenutil.DebugPrint for rendering.
func NewFPSWidget() *Node {
	// 100x32 is enough for "FPS: 60.0\nTPS: 60.0"
	img := ebiten.NewImage(100, 32)

	node := NewSprite("fps_widget", TextureRegion{})
	node.SetCustomImage(img)
	node.RenderLayer = 255 // Draw on top

	var lastUpdate float64

	node.OnUpdate = func(dt float64) {
		lastUpdate += dt
		if lastUpdate < 0.5 {
			return
		}
		lastUpdate = 0

		img.Clear()
		// Semi-transparent background for readability
		img.Fill(color.RGBA{0, 0, 0, 128})

		fps := ebiten.ActualFPS()
		tps := ebiten.ActualTPS()
		ebitenutil.DebugPrint(img, fmt.Sprintf("FPS: %.1f\nTPS: %.1f", fps, tps))
	}

	return node
}
