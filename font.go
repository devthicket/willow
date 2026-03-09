package willow

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/phanxgames/willow/internal/text"
)

// ---------------------------------------------------------------------------
// Type aliases for SDF generation types
// ---------------------------------------------------------------------------

// SDFGenOptions configures SDF atlas generation.
type SDFGenOptions = text.SDFGenOptions

// GlyphBitmap holds a rasterized glyph and its metrics for atlas packing.
type GlyphBitmap = text.GlyphBitmap

// SpriteFont renders text from a pre-generated SDF or MSDF atlas.
// It implements the Font interface for measurement and layout.
type SpriteFont = text.SpriteFont

// ---------------------------------------------------------------------------
// Function aliases for SDF generation and font loading
// ---------------------------------------------------------------------------

// GenerateSDFFromBitmaps creates an SDF atlas from pre-rasterized glyph bitmaps.
// This is the core algorithm used by both runtime and offline generation paths.
var GenerateSDFFromBitmaps = text.GenerateSDFFromBitmaps

// LoadSpriteFont parses SDF font metrics JSON and returns a SpriteFont.
// The pageIndex specifies which atlas page the SDF glyphs are on.
// Register the atlas page image on the Scene via Scene.RegisterPage.
var LoadSpriteFont = text.LoadSpriteFont

// LoadSpriteFontFromTTF generates an SDF font atlas at runtime from TTF/OTF data.
// Uses pure-Go rasterization (golang.org/x/image/font/opentype) so it can be
// called before the Ebitengine game loop starts. Returns the SpriteFont, the atlas
// as an ebiten.Image (caller must register via Scene.RegisterPage), the raw
// atlas as an *image.NRGBA (for saving to disk), and any error.
var LoadSpriteFontFromTTF = text.LoadSpriteFontFromTTF

// ---------------------------------------------------------------------------
// NewFontFromTTF — stays in root (needs atlasManager)
// ---------------------------------------------------------------------------

// NewFontFromTTF generates an SDF font from TTF/OTF data, registers the atlas
// page, and returns the font ready to use:
//
//	font, err := willow.NewFontFromTTF(ttfData, 80)
//	label := willow.NewText("label", "Hello!", font)
//	label.TextBlock.FontSize = 24
//
// Size is the rasterization size in pixels (higher = better quality when scaled up).
// Use 0 for the default (80px).
func NewFontFromTTF(ttfData []byte, size float64) (*SpriteFont, error) {
	am := atlasManager()
	pageIndex := am.AllocPage()
	sdfFont, atlasImg, _, err := LoadSpriteFontFromTTF(ttfData, SDFGenOptions{
		Size:      size,
		PageIndex: uint16(pageIndex),
	})
	if err != nil {
		return nil, err
	}
	am.RegisterPage(pageIndex, atlasImg)
	sdfFont.SetTTFData(ttfData)
	return sdfFont, nil
}

// ---------------------------------------------------------------------------
// System font loading
// ---------------------------------------------------------------------------

// LoadFontFromPathAsTtf reads a TTF/OTF file from disk (project-relative or absolute path).
func LoadFontFromPathAsTtf(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// LoadFontFromSystemAsTtf searches OS system font directories for a font by name
// (e.g. "Arial"). Tries .ttf and .otf extensions. Returns the raw TTF/OTF bytes.
func LoadFontFromSystemAsTtf(name string) ([]byte, error) {
	dirs := systemFontDirs()
	for _, dir := range dirs {
		for _, ext := range []string{".ttf", ".otf"} {
			p := filepath.Join(dir, name+ext)
			data, err := os.ReadFile(p)
			if err == nil {
				return data, nil
			}
		}
	}
	return nil, fmt.Errorf("font %q not found in system font directories", name)
}

func systemFontDirs() []string {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		dirs := []string{
			"/Library/Fonts",
			"/System/Library/Fonts/Supplemental",
		}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
		}
		return dirs
	case "windows":
		return []string{`C:\Windows\Fonts`}
	case "linux":
		return []string{
			"/usr/share/fonts/truetype",
			"/usr/share/fonts/TTF",
			"/usr/share/fonts/opentype",
		}
	default:
		return nil
	}
}

