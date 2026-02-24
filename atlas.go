package willow

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// TextureRegion describes a sub-rectangle within an atlas page.
// Value type (32 bytes)  -  stored directly on Node, no pointer.
type TextureRegion struct {
	Page      uint16 // atlas page index (references Scene.pages)
	X, Y      uint16 // top-left corner of the sub-image rect within the atlas page
	Width     uint16 // width of the sub-image rect (may differ from OriginalW if trimmed)
	Height    uint16 // height of the sub-image rect (may differ from OriginalH if trimmed)
	OriginalW uint16 // untrimmed sprite width as authored
	OriginalH uint16 // untrimmed sprite height as authored
	OffsetX   int16  // horizontal trim offset from TexturePacker
	OffsetY   int16  // vertical trim offset from TexturePacker
	Rotated   bool   // true if the region is stored 90 degrees clockwise in the atlas
}

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
type PackerConfig struct {
	PageWidth  int // atlas page width in pixels (default 2048)
	PageHeight int // atlas page height in pixels (default 2048)
	Padding    int // pixel gap between packed images (default 1); use NoPadding for 0
	noPadding  bool
}

// NoPadding returns a PackerConfig modifier that explicitly sets padding to 0.
// Without this, a zero Padding field is treated as "use default (1)".
func (c PackerConfig) NoPadding() PackerConfig {
	c.Padding = 0
	c.noPadding = true
	return c
}

// normalizePackerConfig fills in defaults for any unset fields.
func normalizePackerConfig(c PackerConfig) PackerConfig {
	if c.PageWidth <= 0 {
		c.PageWidth = 2048
	}
	if c.PageHeight <= 0 {
		c.PageHeight = 2048
	}
	if c.Padding == 0 && !c.noPadding {
		c.Padding = 1
	}
	if c.Padding < 0 {
		c.Padding = 1
	}
	return c
}

// Atlas holds one or more atlas page images and a map of named regions.
type Atlas struct {
	// Pages contains the atlas page images indexed by page number.
	Pages   []*ebiten.Image
	regions map[string]TextureRegion

	packer    *shelfPacker // lazily created on first Add()
	packerCfg PackerConfig // stored config for lazy packer init
}

// Region returns the TextureRegion for the given name.
// If the name doesn't exist, it logs a warning (debug stderr) and returns
// a 1×1 magenta placeholder region on page index magentaPlaceholderPage.
func (a *Atlas) Region(name string) TextureRegion {
	if r, ok := a.regions[name]; ok {
		return r
	}
	if globalDebug {
		log.Printf("willow: atlas region %q not found, using magenta placeholder", name)
	}
	return magentaRegion()
}

// magenta placeholder singleton (no sync.Once  -  willow is single-threaded)
var magentaImage *ebiten.Image

func ensureMagentaImage() *ebiten.Image {
	if magentaImage == nil {
		magentaImage = ebiten.NewImage(1, 1)
		magentaImage.Fill(color.RGBA{R: 255, G: 0, B: 255, A: 255})
	}
	return magentaImage
}

// magentaPlaceholderPage is a sentinel page index used for magenta placeholders.
// It's high enough to never collide with real atlas pages.
const magentaPlaceholderPage = 0xFFFF

func magentaRegion() TextureRegion {
	return TextureRegion{
		Page:      magentaPlaceholderPage,
		X:         0,
		Y:         0,
		Width:     1,
		Height:    1,
		OriginalW: 1,
		OriginalH: 1,
	}
}

// LoadAtlas parses TexturePacker JSON data and associates the given page images.
// Supports both the hash format (single "frames" object) and the array format
// ("textures" array with per-page frame lists).
func LoadAtlas(jsonData []byte, pages []*ebiten.Image) (*Atlas, error) {
	// Probe top-level keys to detect format.
	var probe struct {
		Frames   json.RawMessage `json:"frames"`
		Textures json.RawMessage `json:"textures"`
	}
	if err := json.Unmarshal(jsonData, &probe); err != nil {
		return nil, fmt.Errorf("willow: failed to parse atlas JSON: %w", err)
	}

	atlas := &Atlas{
		Pages:   pages,
		regions: make(map[string]TextureRegion),
	}

	if probe.Textures != nil {
		// Multi-page array format
		if err := parseArrayFormat(probe.Textures, atlas); err != nil {
			return nil, err
		}
	} else if probe.Frames != nil {
		// Single-page hash format
		if err := parseHashFrames(probe.Frames, 0, atlas); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("willow: atlas JSON has neither \"frames\" nor \"textures\" key")
	}

	return atlas, nil
}

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
func NewAtlas(cfg ...PackerConfig) *Atlas {
	var c PackerConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c = normalizePackerConfig(c)
	return &Atlas{
		regions:   make(map[string]TextureRegion),
		packerCfg: c,
	}
}

// ensurePacker lazily creates the shelf packer on first dynamic Add.
func (a *Atlas) ensurePacker() {
	if a.packer != nil {
		return
	}
	cfg := normalizePackerConfig(a.packerCfg)
	a.packer = newShelfPacker(cfg.PageWidth, cfg.PageHeight, cfg.Padding)
}

// Add packs img into the atlas under the given name and returns its
// TextureRegion. If name already exists the existing region is returned
// (idempotent). Returns an error for nil, zero-size, or oversized images.
//
// The source image's pixels are copied onto an atlas page via DrawImage.
// This is a one-time cost per Add call, not per frame. After Add, the
// source image is no longer referenced and may be garbage collected.
//
// Add also works on atlases created by LoadAtlas, allowing you to extend
// a static atlas with dynamically loaded images at runtime. The packer
// allocates new pages beyond the static ones.
func (a *Atlas) Add(name string, img *ebiten.Image) (TextureRegion, error) {
	if r, ok := a.regions[name]; ok {
		return r, nil
	}
	if img == nil {
		return TextureRegion{}, fmt.Errorf("willow: Add(%q): image is nil", name)
	}
	if a.regions == nil {
		a.regions = make(map[string]TextureRegion)
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return TextureRegion{}, fmt.Errorf("willow: cannot add zero-size image %q (%d×%d)", name, w, h)
	}

	a.ensurePacker()

	pp, x, y, err := a.packer.pack(w, h)
	if err != nil {
		return TextureRegion{}, fmt.Errorf("willow: Add(%q): %w", name, err)
	}

	// Copy pixels onto the atlas page at the packed position.
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(x), float64(y))
	pp.img.DrawImage(img, &op)

	// Keep Atlas.Pages in sync so callers (e.g. LightLayer.SetPages) see
	// dynamically allocated pages.
	for len(a.Pages) <= pp.pageIdx {
		a.Pages = append(a.Pages, nil)
	}
	a.Pages[pp.pageIdx] = pp.img

	r := TextureRegion{
		Page:      uint16(pp.pageIdx),
		X:         uint16(x),
		Y:         uint16(y),
		Width:     uint16(w),
		Height:    uint16(h),
		OriginalW: uint16(w),
		OriginalH: uint16(h),
	}
	a.regions[name] = r
	return r, nil
}

// Has reports whether a region with the given name exists in the atlas.
func (a *Atlas) Has(name string) bool {
	_, ok := a.regions[name]
	return ok
}

// RegionCount returns the number of named regions in the atlas.
func (a *Atlas) RegionCount() int {
	return len(a.regions)
}

// Remove deletes the named region from the atlas lookup maps.
// It does not reclaim the pixel space on the atlas page — the pixels
// remain as dead space. Re-adding the same name after Remove packs into
// a new slot; it does not reuse the old one.
func (a *Atlas) Remove(name string) {
	delete(a.regions, name)
}

// --- JSON structure types ---

type jsonRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type jsonSize struct {
	W int `json:"w"`
	H int `json:"h"`
}

type jsonFrame struct {
	Frame            jsonRect `json:"frame"`
	Rotated          bool     `json:"rotated"`
	Trimmed          bool     `json:"trimmed"`
	SpriteSourceSize jsonRect `json:"spriteSourceSize"`
	SourceSize       jsonSize `json:"sourceSize"`
}

type jsonTexturePage struct {
	Image  string               `json:"image"`
	Frames map[string]jsonFrame `json:"frames"`
}

// parseHashFrames parses the hash format: {"name": {frame...}, ...}
func parseHashFrames(raw json.RawMessage, pageIndex uint16, atlas *Atlas) error {
	var frames map[string]jsonFrame
	if err := json.Unmarshal(raw, &frames); err != nil {
		return fmt.Errorf("willow: failed to parse atlas frames: %w", err)
	}
	for name, f := range frames {
		atlas.regions[name] = frameToRegion(f, pageIndex)
	}
	return nil
}

// parseArrayFormat parses the array format: [{"image":"...", "frames":{...}}, ...]
func parseArrayFormat(raw json.RawMessage, atlas *Atlas) error {
	var textures []jsonTexturePage
	if err := json.Unmarshal(raw, &textures); err != nil {
		return fmt.Errorf("willow: failed to parse atlas textures array: %w", err)
	}
	for i, tex := range textures {
		for name, f := range tex.Frames {
			atlas.regions[name] = frameToRegion(f, uint16(i))
		}
	}
	return nil
}

func frameToRegion(f jsonFrame, page uint16) TextureRegion {
	return TextureRegion{
		Page:      page,
		X:         uint16(f.Frame.X),
		Y:         uint16(f.Frame.Y),
		Width:     uint16(f.Frame.W),
		Height:    uint16(f.Frame.H),
		OriginalW: uint16(f.SourceSize.W),
		OriginalH: uint16(f.SourceSize.H),
		OffsetX:   int16(f.SpriteSourceSize.X),
		OffsetY:   int16(f.SpriteSourceSize.Y),
		Rotated:   f.Rotated,
	}
}
