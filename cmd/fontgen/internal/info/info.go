// Package info prints metadata about TTF/OTF files and .fontbundle archives.
package info

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"path/filepath"
	"strings"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// PrintTTF prints metadata about a TTF/OTF file.
func PrintTTF(data []byte, path string) error {
	f, err := sfnt.Parse(data)
	if err != nil {
		return fmt.Errorf("info: %w", err)
	}

	var b sfnt.Buffer

	fam := nameID(f, &b, 16)
	if fam == "" {
		fam = nameID(f, &b, 1)
	}
	sub := nameID(f, &b, 17)
	if sub == "" {
		sub = nameID(f, &b, 2)
	}
	if sub == "" {
		sub = "Regular"
	}

	numGlyphs := f.NumGlyphs()
	upm := f.UnitsPerEm()

	hasKern := probeKerning(f, &b)
	fmt.Printf("Family: %s  Subfamily: %s  Glyphs: %d  Units/em: %d  Has kerning: %s\n",
		fam, sub, numGlyphs, upm, yesNo(hasKern))
	_ = path
	return nil
}

// PrintBundle prints metadata about a .fontbundle archive.
func PrintBundle(data []byte, path string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("info: %w", err)
	}

	pngs := map[string][]byte{}
	jsons := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(rc)
		rc.Close()
		name := f.Name
		switch {
		case strings.HasSuffix(name, ".png"):
			pngs[strings.TrimSuffix(name, ".png")] = buf.Bytes()
		case strings.HasSuffix(name, ".json"):
			jsons[strings.TrimSuffix(name, ".json")] = buf.Bytes()
		}
	}

	fmt.Printf("Bundle: %s\n", filepath.Base(path))

	var entries []entryInfo
	for base := range jsons {
		style, size, ok := parseBase(base)
		if !ok {
			continue
		}
		entries = append(entries, entryInfo{style: style, size: size, base: base})
	}
	sortEntries(entries)

	for _, e := range entries {
		jsonData := jsons[e.base]
		pngData := pngs[e.base]

		var detailed struct {
			Glyphs []struct {
				W int `json:"width"`
				H int `json:"height"`
			} `json:"glyphs"`
		}
		_ = json.Unmarshal(jsonData, &detailed)
		glyphCount := len(detailed.Glyphs)

		var atlasW, atlasH int
		if len(pngData) > 0 {
			cfg, _, err := image.DecodeConfig(bytes.NewReader(pngData))
			if err == nil {
				atlasW, atlasH = cfg.Width, cfg.Height
			}
		}

		filled := ""
		if atlasW > 0 && atlasH > 0 {
			usedPx := 0
			for _, g := range detailed.Glyphs {
				usedPx += g.W * g.H
			}
			totalPx := atlasW * atlasH
			if totalPx > 0 {
				pct := int(float64(usedPx) / float64(totalPx) * 100)
				filled = fmt.Sprintf("  %d%% filled", pct)
			}
		}

		fmt.Printf("  %-12s %4.0fpx   %dx%d   %d glyphs%s\n",
			e.style, e.size, atlasW, atlasH, glyphCount, filled)
	}
	return nil
}

// parseBase splits a bundle entry base name like "regular_64" into style and size.
func parseBase(base string) (style string, size float64, ok bool) {
	idx := strings.LastIndex(base, "_")
	if idx < 0 {
		return "", 0, false
	}
	var s float64
	if _, err := fmt.Sscanf(base[idx+1:], "%f", &s); err != nil {
		return "", 0, false
	}
	return base[:idx], s, true
}

type entryInfo struct {
	style string
	size  float64
	base  string
}

func sortEntries(entries []entryInfo) {
	prio := map[string]int{"regular": 0, "bold": 1, "italic": 2, "bold_italic": 3}
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0; j-- {
			a, b := entries[j-1], entries[j]
			pa, pb := prio[a.style], prio[b.style]
			if pa > pb || (pa == pb && a.size > b.size) {
				entries[j-1], entries[j] = entries[j], entries[j-1]
			}
		}
	}
}

func nameID(f *sfnt.Font, b *sfnt.Buffer, id sfnt.NameID) string {
	v, err := f.Name(b, id)
	if err != nil {
		return ""
	}
	return v
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// probeKerning checks a small set of common kern pairs to detect kern support.
func probeKerning(f *sfnt.Font, b *sfnt.Buffer) bool {
	ppem := fixed.I(64)
	probes := []struct{ r0, r1 rune }{
		{'A', 'V'}, {'T', 'o'}, {'W', 'a'}, {'f', 'i'},
	}
	for _, p := range probes {
		g0, err0 := f.GlyphIndex(b, p.r0)
		g1, err1 := f.GlyphIndex(b, p.r1)
		if err0 != nil || err1 != nil || g0 == 0 || g1 == 0 {
			continue
		}
		k, err := f.Kern(b, g0, g1, ppem, xfont.HintingNone)
		if err == nil && k != 0 {
			return true
		}
	}
	return false
}
