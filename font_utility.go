package willow

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

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
