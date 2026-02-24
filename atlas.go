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

// Atlas holds one or more atlas page images and a map of named regions.
type Atlas struct {
	// Pages contains the atlas page images indexed by page number.
	Pages   []*ebiten.Image
	regions map[string]TextureRegion
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
