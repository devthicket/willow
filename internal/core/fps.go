package core

import (
	"fmt"
	"image/color"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// NewFPSWidget creates a Node that displays FPS and TPS as a screen overlay.
func NewFPSWidget() *node.Node {
	img := ebiten.NewImage(100, 32)
	n := node.NewNode("fps_widget", types.NodeTypeSprite)
	n.CustomImage_ = img
	n.RenderLayer = 255
	var lastUpdate float64
	n.OnUpdate = func(dt float64) {
		lastUpdate += dt
		if lastUpdate < 0.5 {
			return
		}
		lastUpdate = 0
		img.Clear()
		img.Fill(color.RGBA{0, 0, 0, 128})
		fps := ebiten.ActualFPS()
		tps := ebiten.ActualTPS()
		ebitenutil.DebugPrint(img, fmt.Sprintf("FPS: %.1f\nTPS: %.1f", fps, tps))
	}
	return n
}
