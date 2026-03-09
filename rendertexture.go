package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/render"
)

// RenderTexture is a persistent offscreen canvas that can be attached to a
// sprite node via SetCustomImage. Unlike pooled render targets used internally,
// a RenderTexture is owned by the caller and is NOT recycled between frames.
type RenderTexture = render.RenderTexture

// RenderTextureDrawOpts controls how an image or sprite is drawn onto a
// RenderTexture when using the "Colored" draw methods.
type RenderTextureDrawOpts = render.RenderTextureDrawOpts

// NewRenderTexture creates a persistent offscreen canvas of the given size.
var NewRenderTexture = render.NewRenderTexture

// resolvePageImage returns the atlas page image for a region, handling the
// magenta placeholder sentinel. Delegates to the atlas/render layer.
func resolvePageImage(region TextureRegion, pages []*ebiten.Image) *ebiten.Image {
	if region.Page == magentaPlaceholderPage {
		return ensureMagentaImage()
	}
	idx := int(region.Page)
	if idx < len(pages) {
		return pages[idx]
	}
	return nil
}
