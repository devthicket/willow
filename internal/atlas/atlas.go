package atlas

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"

	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// Debug enables debug logging (e.g. missing region warnings).
// Set by the root package.
var Debug bool

// PackerConfig controls the dynamic atlas packer created by New.
// Zero-value fields use defaults: 2048×2048 pages with 1 pixel padding.
// Set Padding to 0 explicitly via NoPadding to disable padding (risk of
// texture bleeding).
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

// NormalizePackerConfig fills in defaults for any unset fields.
func NormalizePackerConfig(c PackerConfig) PackerConfig {
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
	Regions map[string]types.TextureRegion

	packer    *ShelfPacker // lazily created on first Add()
	packerCfg PackerConfig // stored config for lazy packer init

	// Batch mode (NewBatch)
	staged    []StagedEntry       // buffered images for Pack()
	stagedSet map[string]struct{} // tracks staged names for O(1) duplicate check
	packed    bool                // true after Pack() is called
	batch     bool                // true if created via NewBatch
}

// Region returns the TextureRegion for the given name.
// If the name doesn't exist, it logs a warning (debug stderr) and returns
// a 1×1 magenta placeholder region on page index MagentaPlaceholderPage.
func (a *Atlas) Region(name string) types.TextureRegion {
	if r, ok := a.Regions[name]; ok {
		return r
	}
	if Debug {
		log.Printf("willow: atlas region %q not found, using magenta placeholder", name)
	}
	return MagentaRegion()
}

// magenta placeholder singleton (no sync.Once — willow is single-threaded)
var magentaImage *ebiten.Image

// EnsureMagentaImage returns the 1×1 magenta placeholder image, creating it if needed.
func EnsureMagentaImage() *ebiten.Image {
	if magentaImage == nil {
		magentaImage = ebiten.NewImage(1, 1)
		magentaImage.Fill(color.RGBA{R: 255, G: 0, B: 255, A: 255})
	}
	return magentaImage
}

// MagentaPlaceholderPage is a sentinel page index used for magenta placeholders.
// It's high enough to never collide with real atlas pages.
const MagentaPlaceholderPage = 0xFFFF

// MagentaRegion returns a 1×1 placeholder region on the sentinel page.
func MagentaRegion() types.TextureRegion {
	return types.TextureRegion{
		Page:      MagentaPlaceholderPage,
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
		Regions: make(map[string]types.TextureRegion),
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

// New creates an empty Atlas ready for dynamic packing via Add.
// An optional PackerConfig controls page size and padding; defaults are
// 2048×2048 with 1 pixel padding.
func New(cfg ...PackerConfig) *Atlas {
	var c PackerConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c = NormalizePackerConfig(c)
	return &Atlas{
		Regions:   make(map[string]types.TextureRegion),
		packerCfg: c,
	}
}

// NewBatch creates an empty Atlas in batch mode. Use Stage to buffer
// images, then Pack to run MaxRects and finalize. No additions after Pack.
//
// Batch mode packs more efficiently than shelf mode because all images are
// known upfront, allowing optimal placement via the MaxRects algorithm.
func NewBatch(cfg ...PackerConfig) *Atlas {
	var c PackerConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c = NormalizePackerConfig(c)
	return &Atlas{
		Regions:   make(map[string]types.TextureRegion),
		packerCfg: c,
		batch:     true,
	}
}

// ensurePacker lazily creates the shelf packer on first dynamic Add.
func (a *Atlas) ensurePacker() {
	if a.packer != nil {
		return
	}
	cfg := NormalizePackerConfig(a.packerCfg)
	a.packer = NewShelfPacker(cfg.PageWidth, cfg.PageHeight, cfg.Padding)
}

// Add packs img into the atlas under the given name and returns its
// TextureRegion. If name already exists the existing region is returned
// (idempotent). Returns an error for nil, zero-size, or oversized images.
func (a *Atlas) Add(name string, img *ebiten.Image) (types.TextureRegion, error) {
	if a.batch {
		return types.TextureRegion{}, fmt.Errorf("willow: Add(%q): use Stage/Pack on batch atlas", name)
	}
	if r, ok := a.Regions[name]; ok {
		return r, nil
	}
	if img == nil {
		return types.TextureRegion{}, fmt.Errorf("willow: Add(%q): image is nil", name)
	}
	if a.Regions == nil {
		a.Regions = make(map[string]types.TextureRegion)
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return types.TextureRegion{}, fmt.Errorf("willow: cannot add zero-size image %q (%d×%d)", name, w, h)
	}

	a.ensurePacker()

	pp, x, y, err := a.packer.Pack(w, h)
	if err != nil {
		return types.TextureRegion{}, fmt.Errorf("willow: Add(%q): %w", name, err)
	}

	// Copy pixels onto the atlas page at the packed position.
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(x), float64(y))
	pp.img.DrawImage(img, &op)

	// Keep Atlas.Pages in sync so callers see dynamically allocated pages.
	for len(a.Pages) <= pp.pageIdx {
		a.Pages = append(a.Pages, nil)
	}
	a.Pages[pp.pageIdx] = pp.img

	r := types.TextureRegion{
		Page:      uint16(pp.pageIdx),
		X:         uint16(x),
		Y:         uint16(y),
		Width:     uint16(w),
		Height:    uint16(h),
		OriginalW: uint16(w),
		OriginalH: uint16(h),
	}
	a.Regions[name] = r
	return r, nil
}

// Has reports whether a region with the given name exists in the atlas.
func (a *Atlas) Has(name string) bool {
	_, ok := a.Regions[name]
	return ok
}

// RegionCount returns the number of named regions in the atlas.
func (a *Atlas) RegionCount() int {
	return len(a.Regions)
}

// Remove deletes the named region from the atlas lookup maps.
// It does not reclaim the pixel space on the atlas page.
//
// Deprecated: Use Free instead, which also reclaims space for reuse.
func (a *Atlas) Remove(name string) {
	delete(a.Regions, name)
}

// Free removes the named region and reclaims its space for reuse by
// future Add calls. The pixels on the atlas page are cleared to prevent
// stale bleeding. Returns an error if the name doesn't exist or if
// called on a batch atlas.
func (a *Atlas) Free(name string) error {
	if a.batch {
		return fmt.Errorf("willow: Free(%q): not supported on batch atlas", name)
	}
	r, ok := a.Regions[name]
	if !ok {
		return fmt.Errorf("willow: Free(%q): region not found", name)
	}

	// Clear pixels on the atlas page.
	pageIdx := int(r.Page)
	if pageIdx < len(a.Pages) && a.Pages[pageIdx] != nil {
		page := a.Pages[pageIdx]
		sub := page.SubImage(image.Rect(
			int(r.X), int(r.Y),
			int(r.X)+int(r.Width), int(r.Y)+int(r.Height),
		)).(*ebiten.Image)
		sub.Clear()
	}

	// Tell the packer to reclaim the slot.
	if a.packer != nil {
		pw := int(r.Width) + a.packer.Padding
		a.packer.Free(pageIdx, int(r.X), int(r.Y), pw)
	}

	delete(a.Regions, name)
	return nil
}

// Replace swaps the pixels for an existing region in-place. The new image
// must be exactly the same size as the existing region. Returns an error
// if the name doesn't exist, sizes differ, or if called on a batch atlas.
func (a *Atlas) Replace(name string, img *ebiten.Image) error {
	if a.batch {
		return fmt.Errorf("willow: Replace(%q): not supported on batch atlas", name)
	}
	r, ok := a.Regions[name]
	if !ok {
		return fmt.Errorf("willow: Replace(%q): region not found", name)
	}
	if img == nil {
		return fmt.Errorf("willow: Replace(%q): image is nil", name)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w != int(r.Width) || h != int(r.Height) {
		return fmt.Errorf("willow: Replace(%q): size mismatch: region is %d×%d, image is %d×%d (use Update for different sizes)",
			name, r.Width, r.Height, w, h)
	}

	pageIdx := int(r.Page)
	if pageIdx < len(a.Pages) && a.Pages[pageIdx] != nil {
		page := a.Pages[pageIdx]
		// Clear old pixels, then draw new ones.
		sub := page.SubImage(image.Rect(
			int(r.X), int(r.Y),
			int(r.X)+int(r.Width), int(r.Y)+int(r.Height),
		)).(*ebiten.Image)
		sub.Clear()

		var op ebiten.DrawImageOptions
		op.GeoM.Translate(float64(r.X), float64(r.Y))
		page.DrawImage(img, &op)
	}
	return nil
}

// Update replaces the image for an existing region. If the new image is
// the same size, it acts like Replace (in-place pixel swap). If the size
// differs, it frees the old region and adds the new image, returning a
// potentially different TextureRegion.
func (a *Atlas) Update(name string, img *ebiten.Image) (types.TextureRegion, error) {
	if a.batch {
		return types.TextureRegion{}, fmt.Errorf("willow: Update(%q): not supported on batch atlas", name)
	}
	if img == nil {
		return types.TextureRegion{}, fmt.Errorf("willow: Update(%q): image is nil", name)
	}

	r, ok := a.Regions[name]
	if !ok {
		return types.TextureRegion{}, fmt.Errorf("willow: Update(%q): region not found", name)
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	// Same size → in-place replace.
	if w == int(r.Width) && h == int(r.Height) {
		if err := a.Replace(name, img); err != nil {
			return types.TextureRegion{}, err
		}
		return r, nil
	}

	// Different size → free + add.
	if err := a.Free(name); err != nil {
		return types.TextureRegion{}, err
	}
	return a.Add(name, img)
}

// Stage buffers an image for later packing via Pack. Only valid on batch
// atlases (created with NewBatch). Returns an error if already packed,
// image is nil/zero-size, or name is a duplicate.
func (a *Atlas) Stage(name string, img *ebiten.Image) error {
	if !a.batch {
		return fmt.Errorf("willow: Stage(%q): use Add on shelf atlas", name)
	}
	if a.packed {
		return fmt.Errorf("willow: Stage(%q): atlas already packed", name)
	}
	if img == nil {
		return fmt.Errorf("willow: Stage(%q): image is nil", name)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return fmt.Errorf("willow: Stage(%q): zero-size image (%d×%d)", name, w, h)
	}
	// Check for duplicate name.
	if a.stagedSet == nil {
		a.stagedSet = make(map[string]struct{})
	}
	if _, dup := a.stagedSet[name]; dup {
		return fmt.Errorf("willow: Stage(%q): duplicate name", name)
	}
	a.stagedSet[name] = struct{}{}
	a.staged = append(a.staged, StagedEntry{
		Name: name,
		Img:  img,
		W:    w,
		H:    h,
	})
	return nil
}

// Pack runs the MaxRects algorithm on all staged images and copies their
// pixels onto atlas pages. After Pack, Stage returns an error and Region
// returns valid regions.
func (a *Atlas) Pack() error {
	if !a.batch {
		return fmt.Errorf("willow: Pack: not a batch atlas (use Add on shelf atlas)")
	}
	if a.packed {
		return fmt.Errorf("willow: Pack: already packed")
	}

	if len(a.staged) == 0 {
		a.packed = true
		return nil
	}

	cfg := NormalizePackerConfig(a.packerCfg)

	// Sort staged entries largest-area-first for better packing.
	SortStagedByArea(a.staged)

	type pageInfo struct {
		rp  *RectPacker
		img *ebiten.Image
		idx int
	}
	var pages []pageInfo

	allocPage := func() (pageInfo, error) {
		am := GlobalManager()
		idx := am.AllocPage()
		if idx > MaxPackerPageIdx {
			return pageInfo{}, fmt.Errorf("willow: atlas page index %d exceeds maximum %d", idx, MaxPackerPageIdx)
		}
		img := ebiten.NewImage(cfg.PageWidth, cfg.PageHeight)
		am.RegisterPage(idx, img)
		rp := NewRectPacker(cfg.PageWidth, cfg.PageHeight)
		pi := pageInfo{rp: rp, img: img, idx: idx}
		pages = append(pages, pi)
		return pi, nil
	}

	for _, entry := range a.staged {
		pw := entry.W + cfg.Padding
		ph := entry.H + cfg.Padding

		if pw > cfg.PageWidth || ph > cfg.PageHeight {
			return fmt.Errorf("willow: Pack: image %q (%d×%d, padded %d×%d) exceeds page size %d×%d",
				entry.Name, entry.W, entry.H, pw, ph, cfg.PageWidth, cfg.PageHeight)
		}

		placed := false
		for pi := range pages {
			if x, y, ok := pages[pi].rp.TryPlace(pw, ph); ok {
				pages[pi].rp.Place(x, y, pw, ph)
				a.copyAndRegister(entry, pages[pi].img, pages[pi].idx, x, y)
				placed = true
				break
			}
		}

		if !placed {
			pi, err := allocPage()
			if err != nil {
				return err
			}
			x, y, ok := pi.rp.TryPlace(pw, ph)
			if !ok {
				return fmt.Errorf("willow: Pack: image %q does not fit on fresh page", entry.Name)
			}
			pi.rp.Place(x, y, pw, ph)
			a.copyAndRegister(entry, pi.img, pi.idx, x, y)
		}
	}

	a.packed = true

	// Sync Pages slice.
	for _, pi := range pages {
		for len(a.Pages) <= pi.idx {
			a.Pages = append(a.Pages, nil)
		}
		a.Pages[pi.idx] = pi.img
	}

	// Clear staged to free references.
	a.staged = nil
	a.stagedSet = nil
	return nil
}

// copyAndRegister copies an image onto a page and records its region.
func (a *Atlas) copyAndRegister(entry StagedEntry, page *ebiten.Image, pageIdx, x, y int) {
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(x), float64(y))
	page.DrawImage(entry.Img, &op)

	a.Regions[entry.Name] = types.TextureRegion{
		Page:      uint16(pageIdx),
		X:         uint16(x),
		Y:         uint16(y),
		Width:     uint16(entry.W),
		Height:    uint16(entry.H),
		OriginalW: uint16(entry.W),
		OriginalH: uint16(entry.H),
	}
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
func parseHashFrames(raw json.RawMessage, pageIndex uint16, a *Atlas) error {
	var frames map[string]jsonFrame
	if err := json.Unmarshal(raw, &frames); err != nil {
		return fmt.Errorf("willow: failed to parse atlas frames: %w", err)
	}
	for name, f := range frames {
		a.Regions[name] = frameToRegion(f, pageIndex)
	}
	return nil
}

// parseArrayFormat parses the array format: [{"image":"...", "frames":{...}}, ...]
func parseArrayFormat(raw json.RawMessage, a *Atlas) error {
	var textures []jsonTexturePage
	if err := json.Unmarshal(raw, &textures); err != nil {
		return fmt.Errorf("willow: failed to parse atlas textures array: %w", err)
	}
	for i, tex := range textures {
		for name, f := range tex.Frames {
			a.Regions[name] = frameToRegion(f, uint16(i))
		}
	}
	return nil
}

func frameToRegion(f jsonFrame, page uint16) types.TextureRegion {
	return types.TextureRegion{
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
