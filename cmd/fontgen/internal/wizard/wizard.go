// Package wizard implements the interactive fontgen wizard using charmbracelet/huh.
package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/devthicket/willow/cmd/fontgen/internal/bake"
	"github.com/devthicket/willow/cmd/fontgen/internal/discovery"
)

// Config holds the resolved wizard choices before baking.
type Config struct {
	Styles     []bake.StyleData
	BakeSizes  []float64
	Charset    string
	OutputPath string
	Bundle     bool // true = .fontbundle, false = individual files
}

// Run presents the interactive wizard and returns the resolved Config.
func Run(results *discovery.Results) (*Config, error) {
	cfg := &Config{Bundle: true}

	// --- Regular ---
	reg, err := pickVariant("Regular", results.Regular, true)
	if err != nil {
		return nil, err
	}
	if reg == "" {
		return nil, fmt.Errorf("regular font is required")
	}
	regData, err := os.ReadFile(reg)
	if err != nil {
		return nil, err
	}
	cfg.Styles = append(cfg.Styles, bake.StyleData{Label: "regular", TTFData: regData})

	// --- Bold ---
	boldPath, err := pickVariant("Bold", results.Bold, false)
	if err != nil {
		return nil, err
	}
	if boldPath == "synthetic" {
		fmt.Println("  bold: synthetic (not found)")
	} else if boldPath != "" {
		d, err := os.ReadFile(boldPath)
		if err != nil {
			return nil, err
		}
		cfg.Styles = append(cfg.Styles, bake.StyleData{Label: "bold", TTFData: d})
	}

	// --- Italic ---
	italicPath, err := pickVariant("Italic", results.Italic, false)
	if err != nil {
		return nil, err
	}
	if italicPath == "synthetic" {
		fmt.Println("  italic: synthetic (not found)")
	} else if italicPath != "" {
		d, err := os.ReadFile(italicPath)
		if err != nil {
			return nil, err
		}
		cfg.Styles = append(cfg.Styles, bake.StyleData{Label: "italic", TTFData: d})
	}

	// --- Bold-Italic ---
	biPath, err := pickVariant("Bold-Italic", results.BoldItalic, false)
	if err != nil {
		return nil, err
	}
	if biPath == "synthetic" {
		fmt.Println("  bold-italic: synthetic (not found)")
	} else if biPath != "" {
		d, err := os.ReadFile(biPath)
		if err != nil {
			return nil, err
		}
		cfg.Styles = append(cfg.Styles, bake.StyleData{Label: "bold_italic", TTFData: d})
	}

	// --- Bake sizes ---
	cfg.BakeSizes, err = pickSizes()
	if err != nil {
		return nil, err
	}

	// --- Charset ---
	cfg.Charset, err = pickCharset()
	if err != nil {
		return nil, err
	}

	// --- Output format ---
	cfg.Bundle, err = pickOutputFormat()
	if err != nil {
		return nil, err
	}

	// --- Output path ---
	defaultOut := strings.ToLower(results.Family)
	if cfg.Bundle {
		defaultOut += ".fontbundle"
	}
	cfg.OutputPath, err = pickOutputPath(defaultOut, cfg.Bundle)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// pickVariant presents candidate font files for one style slot and returns the
// chosen path. Returns "" if the slot was skipped, "synthetic" if synthetic was chosen.
func pickVariant(label string, candidates []discovery.FontFile, required bool) (string, error) {
	fmt.Printf("\n=== %s ===\n", label)

	var options []huh.Option[string]
	for i, c := range candidates {
		options = append(options, huh.NewOption(
			fmt.Sprintf("[%d] %s", i+1, c.Path),
			c.Path,
		))
	}
	if !required {
		options = append(options, huh.NewOption(
			fmt.Sprintf("[%d] Synthetic: auto generate based on regular", len(options)+1),
			"synthetic",
		))
		options = append(options, huh.NewOption(
			fmt.Sprintf("[%d] Skip: do not include this variant", len(options)+1),
			"",
		))
	}

	if len(options) == 0 {
		if required {
			return "", fmt.Errorf("no %s font found", label)
		}
		return "", nil
	}

	var selected string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(label).
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return selected, nil
}

// pickSizes prompts for bake sizes and returns the parsed slice.
func pickSizes() ([]float64, error) {
	fmt.Println("\n=== Bake sizes ===")
	var sizesStr string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Sizes (comma-separated)").
				Placeholder("64,256").
				Value(&sizesStr),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(sizesStr) == "" {
		return []float64{64, 256}, nil
	}
	return parseSizes(sizesStr)
}

// pickCharset prompts for the glyph charset.
func pickCharset() (string, error) {
	fmt.Println("\n=== Glyph charset ===")
	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Charset").
				Options(
					huh.NewOption("[1] ASCII        (95 glyphs — letters, digits, punctuation)", "ascii"),
					huh.NewOption("[2] Latin        (extended latin — covers most Western European languages)", "latin"),
					huh.NewOption("[3] Custom       (enter a string of characters to include)", "custom"),
					huh.NewOption("[4] File         (path to a .txt file listing glyphs to include)", "file"),
					huh.NewOption("[5] Full unicode (all glyphs in the font — largest atlas)", "full"),
				).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	switch choice {
	case "custom":
		var custom string
		f2 := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Characters (e.g. A-Za-z0-9!?)").Value(&custom),
		))
		if err := f2.Run(); err != nil {
			return "", err
		}
		return custom, nil
	case "file":
		var filePath string
		f2 := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Path to .txt file").Placeholder("glyphs.txt").Value(&filePath),
		))
		if err := f2.Run(); err != nil {
			return "", err
		}
		return "file:" + strings.TrimSpace(filePath), nil
	}
	return choice, nil
}

// pickOutputFormat returns true for bundle, false for individual files.
func pickOutputFormat() (bool, error) {
	fmt.Println("\n=== Output format ===")
	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Format").
				Options(
					huh.NewOption("[1] Bundle — single .fontbundle archive", "bundle"),
					huh.NewOption("[2] Individual files — {style}_{size}.png + .json", "individual"),
				).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	return choice == "bundle", nil
}

// pickOutputPath prompts for the output file path.
func pickOutputPath(defaultPath string, bundle bool) (string, error) {
	fmt.Println("\n=== Output ===")
	var path string
	label := "Output file"
	if !bundle {
		label = "Output directory"
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(label).
				Placeholder(defaultPath).
				Value(&path),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		path = defaultPath
	}

	// Confirm overwrite if file exists.
	if _, err := os.Stat(path); err == nil {
		var overwrite bool
		msg := fmt.Sprintf("%s already exists. Overwrite?", filepath.Base(path))
		cf := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().Title(msg).Value(&overwrite),
		))
		if err := cf.Run(); err != nil {
			return "", err
		}
		if !overwrite {
			return "", fmt.Errorf("aborted")
		}
	}
	return path, nil
}

// parseSizes converts a comma-separated size string like "64,256" to []float64.
func parseSizes(s string) ([]float64, error) {
	parts := strings.Split(s, ",")
	var out []float64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid size %q: %w", p, err)
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return []float64{64, 256}, nil
	}
	return out, nil
}
