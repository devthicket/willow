// Command fontgen bakes TTF/OTF fonts into Signed Distance Field (SDF) atlases,
// supporting multiple styles and sizes bundled into a .fontbundle archive.
//
// Usage (bake):
//
//	fontgen arial --sizes=64,256 -o arial.fontbundle
//	fontgen arial --sizes=64,256 --charset=latin -o arial.fontbundle --auto
//	fontgen /path/to/arial.ttf --sizes=64,256 -o arial.fontbundle
//
// Usage (inspect / extract):
//
//	fontgen arial.ttf --info
//	fontgen arial.fontbundle --info
//	fontgen arial.fontbundle --extract
//	fontgen arial.fontbundle --extract=./output
//
// Flags:
//
//	-o             Output path (.fontbundle or directory for individual files)
//	--sizes        Comma-separated bake sizes (default: 64,256)
//	--charset      ascii|latin|full|"chars"|file:path (default: ascii)
//	--range        Override distanceRange for all sizes (default: size×0.125)
//	--bold         Explicit path to bold TTF
//	--italic       Explicit path to italic TTF
//	--bolditalic   Explicit path to bold-italic TTF
//	--regular      Explicit path to regular TTF (alternative to positional arg)
//	--auto / -y    Headless: skip all prompts, auto-select
//	--watch        Re-bake on TTF file change
//	--info         Print metadata without baking
//	--extract[=d]  Unpack .fontbundle to PNG+JSON files
//	--width        Max atlas width in pixels (default: 1024)
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/font/sfnt"

	"github.com/devthicket/willow/cmd/fontgen/internal/bake"
	"github.com/devthicket/willow/cmd/fontgen/internal/discovery"
	"github.com/devthicket/willow/cmd/fontgen/internal/extract"
	"github.com/devthicket/willow/cmd/fontgen/internal/info"
	"github.com/devthicket/willow/cmd/fontgen/internal/wizard"
)

const banner = `fontgen — Signed Distance Field (SDF) Font Generator
Takes a .ttf/.otf file and bakes a PNG atlas + JSON metadata.
Output can be individual files or bundled into a .fontbundle archive.
─────────────────────────────────────────────────────────────────────`

func main() {
	var (
		outPath    = flag.String("o", "", "output path (.fontbundle or directory)")
		sizes      = flag.String("sizes", "64,256", "comma-separated bake sizes")
		charset    = flag.String("charset", "ascii", "charset: ascii|latin|full|\"chars\"|file:path")
		distRange  = flag.Float64("range", 0, "override distanceRange for all sizes (default: size×0.125)")
		boldPath   = flag.String("bold", "", "explicit path to bold TTF/OTF")
		italicPath = flag.String("italic", "", "explicit path to italic TTF/OTF")
		biPath     = flag.String("bolditalic", "", "explicit path to bold-italic TTF/OTF")
		regPath    = flag.String("regular", "", "explicit path to regular TTF/OTF")
		auto       = flag.Bool("auto", false, "headless: skip all prompts")
		autoShort  = flag.Bool("y", false, "headless: skip all prompts (shorthand)")
		watchMode  = flag.Bool("watch", false, "re-bake on TTF file change")
		infoMode   = flag.Bool("info", false, "print metadata without baking")
		width      = flag.Int("width", 1024, "max atlas width in pixels")
	)

	// --extract is special: it can be a boolean or have an optional value.
	extractDir := flag.String("extract", "\x00", "unpack .fontbundle to files (optional dir)")
	flag.Parse()

	headless := *auto || *autoShort

	// Resolve positional arg (font name or path).
	arg := ""
	if flag.NArg() > 0 {
		arg = flag.Arg(0)
	}

	// --info mode.
	if *infoMode {
		if arg == "" {
			fatalf("--info requires a font name, TTF/OTF path, or .fontbundle path\n")
		}
		runInfo(arg)
		return
	}

	// --extract mode.
	if *extractDir != "\x00" {
		if arg == "" {
			fatalf("--extract requires a .fontbundle path as argument\n")
		}
		if !strings.HasSuffix(strings.ToLower(arg), ".fontbundle") {
			fatalf("--extract: %s does not look like a .fontbundle file\n", arg)
		}
		data, err := os.ReadFile(arg)
		check(err)
		dir := *extractDir
		if dir == "" {
			dir = "."
		}
		fmt.Printf("Extracting %s → %s\n", filepath.Base(arg), dir)
		check(extract.Bundle(data, dir))
		return
	}

	// Bake mode: need a font source.
	if arg == "" && *regPath == "" {
		fmt.Println(banner)
		fatalf("usage: fontgen <name|path> [flags]\nRun fontgen --help for options.\n")
	}

	// Print banner for interactive/bake modes.
	if !headless {
		fmt.Println(banner)
	}

	// Resolve the font sources.
	styles, watchPaths, family, err := resolveStyles(arg, *regPath, *boldPath, *italicPath, *biPath, headless)
	check(err)

	// Parse bake options.
	bakeSizes, err := parseSizesFlag(*sizes)
	check(err)

	charsetStr, err := resolveCharset(*charset, styles[0].TTFData)
	check(err)

	opts := bake.Options{
		BakeSizes:     bakeSizes,
		Charset:       charsetStr,
		AtlasWidth:    *width,
		DistanceRange: *distRange,
	}

	// Interactive wizard.
	if !headless {
		cfg, err := runWizard(arg, *regPath, *boldPath, *italicPath, *biPath)
		if err != nil {
			fatalf("%v\n", err)
		}
		styles = cfg.Styles
		opts.BakeSizes = cfg.BakeSizes
		opts.Charset, err = resolveCharset(cfg.Charset, styles[0].TTFData)
		check(err)
		if *outPath == "" {
			*outPath = cfg.OutputPath
		}
	}

	if *outPath == "" {
		*outPath = strings.ToLower(family) + ".fontbundle"
	}

	// Print summary.
	printSummary(styles, opts, *outPath)

	// Confirm overwrite in interactive mode.
	if !headless {
		if _, err := os.Stat(*outPath); err == nil {
			var ans string
			fmt.Printf("%s already exists. Overwrite? [Y/n]: ", filepath.Base(*outPath))
			fmt.Scanln(&ans)
			if ans != "" && strings.ToLower(ans) != "y" {
				fmt.Println("aborted")
				return
			}
		}
	}

	// Bake.
	runBake(styles, opts, *outPath, *watchMode, watchPaths)
}

// resolveStyles loads TTF data for each requested style.
func resolveStyles(arg, regPath, boldPath, italicPath, biPath string, headless bool) ([]bake.StyleData, []string, string, error) {
	// Resolve the regular font.
	var regData []byte
	var family string
	var err error

	if regPath != "" {
		regData, err = os.ReadFile(regPath)
		if err != nil {
			return nil, nil, "", err
		}
		family = familyName(regData, strings.TrimSuffix(filepath.Base(regPath), filepath.Ext(regPath)))
	} else if arg != "" {
		if isFilePath(arg) {
			regData, err = os.ReadFile(arg)
			if err != nil {
				return nil, nil, "", err
			}
			family = familyName(regData, strings.TrimSuffix(filepath.Base(arg), filepath.Ext(arg)))
		} else {
			// Name-based discovery.
			results, err := discovery.ByName(arg)
			if err != nil {
				return nil, nil, "", err
			}
			family = arg
			if len(results.Regular) == 0 {
				return nil, nil, "", fmt.Errorf("no font found for %q — try passing a file path directly", arg)
			}
			regPath = results.Regular[0].Path
			regData, err = os.ReadFile(regPath)
			if err != nil {
				return nil, nil, "", err
			}
			// In headless mode, auto-fill variants from discovery.
			if headless && boldPath == "" && len(results.Bold) > 0 {
				boldPath = results.Bold[0].Path
			}
			if headless && italicPath == "" && len(results.Italic) > 0 {
				italicPath = results.Italic[0].Path
			}
			if headless && biPath == "" && len(results.BoldItalic) > 0 {
				biPath = results.BoldItalic[0].Path
			}
		}
	}

	styles := []bake.StyleData{{Label: "regular", TTFData: regData}}
	watchPaths := []string{regPath}

	for _, v := range []struct {
		label string
		path  string
	}{
		{"bold", boldPath},
		{"italic", italicPath},
		{"bold_italic", biPath},
	} {
		if v.path == "" {
			continue
		}
		d, err := os.ReadFile(v.path)
		if err != nil {
			return nil, nil, "", fmt.Errorf("%s: %w", v.label, err)
		}
		styles = append(styles, bake.StyleData{Label: v.label, TTFData: d})
		watchPaths = append(watchPaths, v.path)
	}

	return styles, watchPaths, family, nil
}

// runWizard runs the interactive wizard if not in headless mode.
func runWizard(arg, regPath, boldPath, italicPath, biPath string) (*wizard.Config, error) {
	var results *discovery.Results
	var err error

	if isFilePath(arg) || regPath != "" {
		path := regPath
		if path == "" {
			path = arg
		}
		results, err = discovery.ByPath(path)
	} else {
		results, err = discovery.ByName(arg)
	}
	if err != nil || results == nil {
		results = &discovery.Results{Family: arg}
	}

	// Override discovered candidates with explicit flags.
	if boldPath != "" {
		results.Bold = []discovery.FontFile{{Path: boldPath}}
	}
	if italicPath != "" {
		results.Italic = []discovery.FontFile{{Path: italicPath}}
	}
	if biPath != "" {
		results.BoldItalic = []discovery.FontFile{{Path: biPath}}
	}

	return wizard.Run(results)
}

// runBake executes the bake (with optional watch loop).
func runBake(styles []bake.StyleData, opts bake.Options, outPath string, watchMode bool, watchPaths []string) {
	doBake := func() {
		fmt.Printf("Writing %s...\n", outPath)
		bundleData, err := bake.Bundle(styles, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		if err := os.WriteFile(outPath, bundleData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
			return
		}
		fmt.Printf("done\n")
	}

	doBake()

	if !watchMode {
		return
	}

	// Watch mode: poll for file changes.
	fmt.Printf("[watching %d file(s) — Ctrl+C to stop]\n", len(watchPaths))
	mtimes := fileMtimes(watchPaths)
	for {
		time.Sleep(500 * time.Millisecond)
		cur := fileMtimes(watchPaths)
		changed := false
		for p, t := range cur {
			if mtimes[p] != t {
				changed = true
				break
			}
		}
		if changed {
			fmt.Printf("[%s] change detected, rebaking...\n", time.Now().Format("2006-01-02 15:04:05"))
			doBake()
			mtimes = fileMtimes(watchPaths)
		}
	}
}

// runInfo prints metadata for a TTF or .fontbundle file.
func runInfo(path string) {
	data, err := os.ReadFile(path)
	check(err)
	if strings.HasSuffix(strings.ToLower(path), ".fontbundle") {
		check(info.PrintBundle(data, path))
	} else {
		check(info.PrintTTF(data, path))
	}
}

func printSummary(styles []bake.StyleData, opts bake.Options, outPath string) {
	var sizeStrs []string
	for _, s := range opts.BakeSizes {
		sizeStrs = append(sizeStrs, fmt.Sprintf("%.0f (distanceRange %.0f)", s, s*0.125))
	}
	charsetLabel := opts.Charset
	if len(charsetLabel) > 40 {
		charsetLabel = fmt.Sprintf("%d chars", len([]rune(charsetLabel)))
	}
	fmt.Println()
	for _, s := range styles {
		fmt.Printf("  %-12s baked\n", s.Label)
	}
	fmt.Printf("\nCharset:      %s\n", charsetLabel)
	fmt.Printf("Bake sizes:   %s\n", strings.Join(sizeStrs, ", "))
	fmt.Printf("Writing %s\n", outPath)
}

// resolveCharset converts a charset flag value to a rune string.
func resolveCharset(spec string, regularTTF []byte) (string, error) {
	if strings.HasPrefix(spec, "file:") {
		path := strings.TrimPrefix(spec, "file:")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("charset file: %w", err)
		}
		return string(data), nil
	}
	switch spec {
	case "full":
		return bake.FullCharset(regularTTF)
	default:
		return bake.CharsetFor(spec), nil
	}
}

func parseSizesFlag(s string) ([]float64, error) {
	var out []float64
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var v float64
		if _, err := fmt.Sscanf(part, "%f", &v); err != nil {
			return nil, fmt.Errorf("invalid size %q", part)
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return []float64{64, 256}, nil
	}
	return out, nil
}

func familyName(ttfData []byte, fallback string) string {
	f, err := sfnt.Parse(ttfData)
	if err != nil {
		return fallback
	}
	var b sfnt.Buffer
	name, err := f.Name(&b, 16) // NameIDPreferredFamily
	if err != nil || name == "" {
		name, _ = f.Name(&b, 1) // NameIDFamily
	}
	if name == "" {
		return fallback
	}
	return name
}

func isFilePath(s string) bool {
	if strings.Contains(s, "/") || strings.Contains(s, `\`) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(s))
	return ext == ".ttf" || ext == ".otf" || ext == ".fontbundle"
}

func fileMtimes(paths []string) map[string]int64 {
	m := make(map[string]int64, len(paths))
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil {
			m[p] = fi.ModTime().UnixNano()
		}
	}
	return m
}

func check(err error) {
	if err != nil {
		fatalf("%v\n", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format, args...)
	os.Exit(1)
}
