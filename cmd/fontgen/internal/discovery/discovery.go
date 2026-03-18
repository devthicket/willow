// Package discovery searches system font directories for TTF/OTF files
// and groups them by family and style using OpenType name table metadata.
package discovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/image/font/sfnt"
)

// Style identifies a font variant slot.
type Style int

const (
	StyleRegular Style = iota
	StyleBold
	StyleItalic
	StyleBoldItalic
)

// FontFile describes a font file found on the system.
type FontFile struct {
	Path      string
	Family    string
	Subfamily string
	Style     Style
}

// Results groups candidate font files by style slot for a given family.
type Results struct {
	Family     string
	Regular    []FontFile
	Bold       []FontFile
	Italic     []FontFile
	BoldItalic []FontFile
}

// ByName searches all font directories for fonts matching the given family name
// and returns candidates grouped by style.
func ByName(family string) (*Results, error) {
	dirs := searchDirs()
	return searchDirsResults(family, dirs)
}

// ByPath loads a specific font file as the regular slot and discovers variants
// from the same directory and system font directories.
func ByPath(path string) (*Results, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}

	var b sfnt.Buffer
	family := nameID(f, &b, 16)
	if family == "" {
		family = nameID(f, &b, 1)
	}
	if family == "" {
		family = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	// Start with the provided file as regular.
	primary := FontFile{
		Path:      path,
		Family:    family,
		Subfamily: "Regular",
		Style:     StyleRegular,
	}

	// Discover variants from the same directory plus system dirs.
	dirs := append([]string{filepath.Dir(path)}, searchDirs()...)
	results, err := searchDirsResults(family, dirs)
	if err != nil {
		results = &Results{Family: family}
	}

	// The provided path is always the regular slot.
	results.Regular = []FontFile{primary}
	return results, nil
}

// searchDirsResults walks the given directories and groups matching fonts.
func searchDirsResults(family string, dirs []string) (*Results, error) {
	famLower := strings.ToLower(family)
	r := &Results{Family: family}

	seen := map[string]bool{}
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".ttf" && ext != ".otf" {
				return nil
			}
			abs, _ := filepath.Abs(path)
			if seen[abs] {
				return nil
			}
			seen[abs] = true

			ff, ok := parseFont(abs, famLower)
			if !ok {
				return nil
			}
			switch ff.Style {
			case StyleRegular:
				r.Regular = append(r.Regular, ff)
			case StyleBold:
				r.Bold = append(r.Bold, ff)
			case StyleItalic:
				r.Italic = append(r.Italic, ff)
			case StyleBoldItalic:
				r.BoldItalic = append(r.BoldItalic, ff)
			}
			return nil
		})
	}
	return r, nil
}

// parseFont reads a font file and returns a FontFile if the family matches.
func parseFont(path, famLower string) (FontFile, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FontFile{}, false
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		return FontFile{}, false
	}

	var b sfnt.Buffer
	fam := nameID(f, &b, 16)
	if fam == "" {
		fam = nameID(f, &b, 1)
	}
	if strings.ToLower(fam) != famLower {
		return FontFile{}, false
	}

	sub := nameID(f, &b, 17)
	if sub == "" {
		sub = nameID(f, &b, 2)
	}

	return FontFile{
		Path:      path,
		Family:    fam,
		Subfamily: sub,
		Style:     classifySubfamily(sub),
	}, true
}

// classifySubfamily maps a subfamily string to a Style slot.
func classifySubfamily(sub string) Style {
	s := strings.ToLower(sub)
	boldItalic := (strings.Contains(s, "bold") && (strings.Contains(s, "italic") || strings.Contains(s, "oblique"))) ||
		s == "bolditalic" || s == "bold italic"
	if boldItalic {
		return StyleBoldItalic
	}
	if strings.Contains(s, "bold") {
		return StyleBold
	}
	if strings.Contains(s, "italic") || strings.Contains(s, "oblique") {
		return StyleItalic
	}
	return StyleRegular
}

// nameID reads a name-table string by ID from an sfnt font.
// It silently ignores errors (missing entries return "").
func nameID(f *sfnt.Font, b *sfnt.Buffer, id sfnt.NameID) string {
	v, err := f.Name(b, id)
	if err != nil {
		return ""
	}
	return v
}

// searchDirs returns the ordered list of directories to search.
func searchDirs() []string {
	var dirs []string

	// 1. Walk current directory downward.
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}

	// 2. Walk up to filesystem root.
	if cwd, err := os.Getwd(); err == nil {
		dir := filepath.Dir(cwd)
		for dir != cwd {
			dirs = append(dirs, dir)
			cwd = dir
			dir = filepath.Dir(cwd)
		}
	}

	// 3. OS system font directories.
	dirs = append(dirs, systemFontDirs()...)

	// 4. Documents and Downloads.
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Documents"))
		dirs = append(dirs, filepath.Join(home, "Downloads"))
	}

	return dirs
}

// systemFontDirs returns platform-specific system font directories.
func systemFontDirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, "Library", "Fonts"),
			"/Library/Fonts",
			"/System/Library/Fonts",
		}
	case "linux":
		return []string{
			filepath.Join(home, ".fonts"),
			filepath.Join(home, ".local", "share", "fonts"),
			"/usr/share/fonts",
			"/usr/local/share/fonts",
		}
	case "windows":
		appdata := os.Getenv("USERPROFILE")
		return []string{
			filepath.Join(appdata, "AppData", "Local", "Microsoft", "Windows", "Fonts"),
			`C:\Windows\Fonts`,
		}
	}
	return nil
}
