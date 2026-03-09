package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/lighting"
	"github.com/phanxgames/willow/internal/render"
)

// Light represents a light source in a LightLayer.
// Aliased from internal/lighting.Light.
type Light = lighting.Light

// LightLayer provides a convenient 2D lighting effect using erase blending.
// Aliased from internal/lighting.LightLayer.
type LightLayer = lighting.LightLayer

// NewLightLayer creates a light layer covering (w x h) pixels.
// ambientAlpha controls the base darkness (0 = fully transparent, 1 = fully opaque black).
var NewLightLayer = lighting.NewLightLayer

// generateCircle creates a feathered white circle image with the given radius.
// Delegates to lighting.GenerateCircle.
var generateCircle = lighting.GenerateCircle

func init() {
	// Wire lighting function pointers.
	lighting.NewRenderTextureFn = func(w, h int) any {
		return render.NewRenderTexture(w, h)
	}
	lighting.RenderTextureImageFn = func(rt any) *ebiten.Image {
		return rt.(*render.RenderTexture).Image()
	}
	lighting.RenderTextureNewSpriteFn = func(rt any, name string) *Node {
		return rt.(*render.RenderTexture).NewSpriteNode(name)
	}
	lighting.RenderTextureDisposeFn = func(rt any) {
		rt.(*render.RenderTexture).Dispose()
	}
	lighting.EnsureMagentaImageFn = ensureMagentaImage
	lighting.Clamp01Fn = clamp01
}
