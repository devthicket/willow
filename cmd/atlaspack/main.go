// Command atlaspack packs PNG sprites into a texture atlas and outputs a
// Willow-compatible .png + .json pair (TexturePacker hash format).
//
// Usage:
//
//	atlaspack [flags] <paths...>
//
// Positional args are auto-detected:
//   - Directory → recursive scan for .png files
//   - Contains * → glob expansion
//   - Ends in .json → parsed as manifest (flags encoded as JSON)
//   - Ends in .png → explicit file
//
// Multiple types can be mixed:
//
//	atlaspack -out atlas sprites/ extra/boss.png overlay/*.png
//
// Flags:
//
//	-out        Output prefix → <out>.png + <out>.json (default "atlas")
//	-width      Atlas width in pixels (default 1024)
//	-height     Atlas height in pixels (default 1024)
//	-padding    Pixels between sprites (default 1)
//	-no-trim    Disable transparent whitespace trimming
//	-no-rotate  Disable 90° CW rotation for tighter packing
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// SpriteInput pairs a file path with the sprite name derived from it.
type SpriteInput struct {
	FilePath   string
	SpriteName string
}

// config holds the resolved CLI/manifest flags.
type config struct {
	Out     string `json:"out"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Padding int    `json:"padding"`
	Trim    bool   `json:"trim"`
	Rotate  bool   `json:"rotate"`
	paths   []string
}

func main() {
	var cfg config

	out := flag.String("out", "atlas", "output file prefix (.png and .json)")
	width := flag.Int("width", 1024, "atlas width in pixels")
	height := flag.Int("height", 1024, "atlas height in pixels")
	padding := flag.Int("padding", 1, "pixels between sprites")
	noTrim := flag.Bool("no-trim", false, "disable transparent whitespace trimming")
	noRotate := flag.Bool("no-rotate", false, "disable 90° CW rotation")
	flag.Parse()

	cfg = config{
		Out:     *out,
		Width:   *width,
		Height:  *height,
		Padding: *padding,
		Trim:    !*noTrim,
		Rotate:  !*noRotate,
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: atlaspack [flags] <paths...>")
		fmt.Fprintln(os.Stderr, "  paths: directories, .png files, globs (*.png), or .json manifests")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Track which flags were explicitly set on the command line.
	explicit := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { explicit[f.Name] = true })

	// Resolve inputs: separate manifests from other paths.
	var directPaths []string
	for _, arg := range args {
		if strings.HasSuffix(arg, ".json") {
			mcfg, mpaths, err := loadManifest(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "loading manifest %s: %v\n", arg, err)
				os.Exit(1)
			}
			// Manifest values are defaults; CLI flags override.
			if !explicit["out"] && mcfg.Out != "" {
				cfg.Out = mcfg.Out
			}
			if !explicit["width"] && mcfg.Width != 0 {
				cfg.Width = mcfg.Width
			}
			if !explicit["height"] && mcfg.Height != 0 {
				cfg.Height = mcfg.Height
			}
			if !explicit["padding"] && mcfg.Padding != 0 {
				cfg.Padding = mcfg.Padding
			}
			if !explicit["no-trim"] {
				cfg.Trim = mcfg.Trim
			}
			if !explicit["no-rotate"] {
				cfg.Rotate = mcfg.Rotate
			}
			directPaths = append(directPaths, mpaths...)
		} else {
			directPaths = append(directPaths, arg)
		}
	}

	inputs, err := resolveInputs(directPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(inputs) == 0 {
		fmt.Fprintln(os.Stderr, "no .png files found in provided paths")
		os.Exit(1)
	}

	// Check for duplicate sprite names.
	if err := checkDuplicates(inputs); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Load and optionally trim.
	sprites, err := loadAndTrim(inputs, cfg.Trim)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Pack.
	packInputs := make([]PackInput, len(sprites))
	for i, s := range sprites {
		packInputs[i] = PackInput{
			ID:     i,
			Width:  s.Image.Bounds().Dx(),
			Height: s.Image.Bounds().Dy(),
		}
	}
	packer := NewMaxRectsPacker(cfg.Width, cfg.Height, cfg.Padding, cfg.Rotate)
	results, err := packer.Pack(packInputs)
	if err != nil {
		// Replace generic ID in error with sprite name.
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Apply pack results to sprites.
	for _, r := range results {
		sprites[r.ID].PackX = r.X
		sprites[r.ID].PackY = r.Y
		sprites[r.ID].Rotated = r.Rotated
	}

	// Compose and write.
	atlasImg := ComposeAtlas(sprites, cfg.Width, cfg.Height)

	pngPath := cfg.Out + ".png"
	if err := writePNG(pngPath, atlasImg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	jsonPath := cfg.Out + ".json"
	jsonData, err := GenerateJSON(sprites, cfg.Width, cfg.Height, pngPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", jsonPath, err)
		os.Exit(1)
	}

	fmt.Printf("Packed %d sprites into %s (%dx%d) + %s\n",
		len(sprites), pngPath, cfg.Width, cfg.Height, jsonPath)
}

// loadManifest parses a JSON manifest file and returns its config values and
// raw path entries (which still need auto-detection resolution).
func loadManifest(path string) (config, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, nil, err
	}
	var m struct {
		Out     string   `json:"out"`
		Width   int      `json:"width"`
		Height  int      `json:"height"`
		Padding int      `json:"padding"`
		Trim    *bool    `json:"trim"`
		Rotate  *bool    `json:"rotate"`
		Paths   []string `json:"paths"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return config{}, nil, err
	}

	cfg := config{
		Out:     m.Out,
		Width:   m.Width,
		Height:  m.Height,
		Padding: m.Padding,
		Trim:    true,
		Rotate:  true,
	}
	if m.Trim != nil {
		cfg.Trim = *m.Trim
	}
	if m.Rotate != nil {
		cfg.Rotate = *m.Rotate
	}

	// Resolve manifest paths relative to the manifest's directory.
	dir := filepath.Dir(path)
	resolved := make([]string, len(m.Paths))
	for i, p := range m.Paths {
		if filepath.IsAbs(p) {
			resolved[i] = p
		} else {
			resolved[i] = filepath.Join(dir, p)
		}
	}

	return cfg, resolved, nil
}

// resolveInputs takes path arguments and resolves them into sprite inputs.
// Directories are walked recursively, globs are expanded, .png files are used directly.
func resolveInputs(paths []string) ([]SpriteInput, error) {
	var inputs []SpriteInput

	for _, p := range paths {
		if strings.Contains(p, "*") {
			// Glob pattern.
			matches, err := filepath.Glob(p)
			if err != nil {
				return nil, fmt.Errorf("invalid glob %q: %w", p, err)
			}
			for _, m := range matches {
				if !strings.HasSuffix(strings.ToLower(m), ".png") {
					continue
				}
				name := strings.TrimSuffix(filepath.Base(m), filepath.Ext(m))
				inputs = append(inputs, SpriteInput{FilePath: m, SpriteName: name})
			}
		} else if info, err := os.Stat(p); err == nil && info.IsDir() {
			// Directory — walk recursively.
			base := p
			err := filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".png") {
					return nil
				}
				rel, _ := filepath.Rel(base, path)
				name := strings.TrimSuffix(rel, filepath.Ext(rel))
				name = filepath.ToSlash(name)
				inputs = append(inputs, SpriteInput{FilePath: path, SpriteName: name})
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walking %s: %w", p, err)
			}
		} else if strings.HasSuffix(strings.ToLower(p), ".png") {
			// Explicit PNG file.
			name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
			inputs = append(inputs, SpriteInput{FilePath: p, SpriteName: name})
		} else {
			return nil, fmt.Errorf("unrecognized path %q (not a directory, .png, or glob)", p)
		}
	}

	return inputs, nil
}

func checkDuplicates(inputs []SpriteInput) error {
	seen := make(map[string]string, len(inputs))
	for _, in := range inputs {
		if prev, ok := seen[in.SpriteName]; ok {
			return fmt.Errorf("duplicate sprite name %q: %s and %s", in.SpriteName, prev, in.FilePath)
		}
		seen[in.SpriteName] = in.FilePath
	}
	return nil
}

func loadAndTrim(inputs []SpriteInput, trim bool) ([]Sprite, error) {
	sprites := make([]Sprite, len(inputs))
	for i, in := range inputs {
		img, err := loadPNG(in.FilePath)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", in.FilePath, err)
		}

		if trim {
			tr := TrimAlpha(img)
			sprites[i] = Sprite{
				Name:      in.SpriteName,
				Image:     tr.Cropped,
				Trimmed:   tr.Trimmed,
				OffsetX:   tr.OffsetX,
				OffsetY:   tr.OffsetY,
				OriginalW: tr.OriginalW,
				OriginalH: tr.OriginalH,
			}
		} else {
			b := img.Bounds()
			sprites[i] = Sprite{
				Name:      in.SpriteName,
				Image:     img,
				OriginalW: b.Dx(),
				OriginalH: b.Dy(),
			}
		}
	}
	return sprites, nil
}

func loadPNG(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	// Convert to NRGBA if needed.
	if nrgba, ok := img.(*image.NRGBA); ok {
		return nrgba, nil
	}
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba.Set(x, y, img.At(x, y))
		}
	}
	return nrgba, nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return f.Close()
}
