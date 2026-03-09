package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/atlas"
)

// PackerConfig controls the dynamic atlas packer created by NewAtlas.
// Zero-value fields use defaults: 2048×2048 pages with 1 pixel padding.
// Set Padding to 0 explicitly via NoPadding to disable padding (risk of
// texture bleeding).
//
// Page size guidance:
//   - Each page allocates pageW × pageH × 4 bytes of VRAM (e.g. 2048×2048 = 16 MB).
//   - Use smaller pages (256–512) when packing a handful of small images.
//   - Use larger pages (1024–2048) when packing many images that must batch together.
//   - All images on the same page are rendered in one DrawTriangles32 submission.
type PackerConfig = atlas.PackerConfig

// Atlas holds one or more atlas page images and a map of named regions.
type Atlas = atlas.Atlas

// NewAtlas creates an empty Atlas ready for dynamic packing via Add.
// An optional PackerConfig controls page size and padding; defaults are
// 2048×2048 with 1 pixel padding.
//
// This is an optional optimization tool. For a small number of runtime
// images (1–5), SetCustomImage or RegisterPage work fine. Use NewAtlas +
// Add when you have many runtime-loaded images (10+) that should batch
// together on the same atlas page.
//
// Ebitengine has its own internal atlas that automatically packs small
// images. NewAtlas operates at a higher level: it controls Willow's batch
// key, which determines how sprites are grouped into DrawTriangles32
// submissions. Images on the same Willow atlas page are guaranteed to be
// one batch (assuming same blend mode and shader).
var NewAtlas = atlas.New

// NewBatchAtlas creates an empty Atlas in batch mode. Use Stage to buffer
// images, then Pack to run MaxRects and finalize. No additions after Pack.
//
// Batch mode packs more efficiently than shelf mode because all images are
// known upfront, allowing optimal placement via the MaxRects algorithm.
var NewBatchAtlas = atlas.NewBatch

// LoadAtlas parses TexturePacker JSON data and associates the given page images.
// Supports both the hash format (single "frames" object) and the array format
// ("textures" array with per-page frame lists).
var LoadAtlas = atlas.LoadAtlas

// magentaPlaceholderPage is re-exported for use in root helpers that reference it.
const magentaPlaceholderPage = atlas.MagentaPlaceholderPage

// ensureMagentaImage returns the 1×1 magenta placeholder image.
func ensureMagentaImage() *ebiten.Image {
	return atlas.EnsureMagentaImage()
}

// magentaRegion returns a placeholder region on the sentinel page.
func magentaRegion() TextureRegion {
	return atlas.MagentaRegion()
}
