// physics-parity diffs two screenshot directories pixel-by-pixel and reports
// the percentage of pixels that differ for each matching filename.
//
// Workflow for the physics parity test:
//
//  1. Run the hand-rolled demo with the parity script, then move/rename the
//     captured PNGs into a baseline directory:
//
//     go run ./examples/demos/physics_handrolled \
//         --autotest=examples/tests/physics-parity.json
//     mv screenshots /tmp/physics-baseline
//
//  2. Run the willow.X demo with the same script:
//
//     go run ./examples/demos/physics \
//         --autotest=examples/tests/physics-parity.json
//     mv screenshots /tmp/physics-candidate
//
//  3. Diff:
//
//     go run ./cmd/physics-parity \
//         --baseline=/tmp/physics-baseline \
//         --candidate=/tmp/physics-candidate
//
// Reports per-PNG delta. With pool-disabled and identical seed, expect zero
// diff. With pool-enabled, allow up to ~0.5% pixel delta on the rest frame.
//
// PNGs are matched by their trailing label (everything after the timestamp
// prefix Scene.Screenshot writes), so timestamp drift between runs is fine.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var timestampPrefix = regexp.MustCompile(`^\d{8}_\d{6}_`)

func main() {
	baseline := flag.String("baseline", "", "directory of baseline PNGs (hand-rolled demo)")
	candidate := flag.String("candidate", "", "directory of candidate PNGs (willow.X demo)")
	threshold := flag.Float64("threshold", 0.5, "pass if every pair has pixel-delta below this percent")
	flag.Parse()

	if *baseline == "" || *candidate == "" {
		fmt.Fprintln(os.Stderr, "both --baseline and --candidate are required")
		os.Exit(2)
	}

	baseMap, err := indexByLabel(*baseline)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	candMap, err := indexByLabel(*candidate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	labels := make([]string, 0, len(baseMap))
	for k := range baseMap {
		labels = append(labels, k)
	}
	sort.Strings(labels)

	worst := 0.0
	missing := 0
	for _, label := range labels {
		basePath := baseMap[label]
		candPath, ok := candMap[label]
		if !ok {
			fmt.Printf("[MISS]  %s: no matching candidate\n", label)
			missing++
			continue
		}
		pct, err := diffPNG(basePath, candPath)
		if err != nil {
			fmt.Printf("[ERR ]  %s: %v\n", label, err)
			missing++
			continue
		}
		status := "OK  "
		if pct > *threshold {
			status = "FAIL"
		}
		fmt.Printf("[%s]  %s: %.4f%% pixels differ\n", status, label, pct)
		if pct > worst {
			worst = pct
		}
	}

	fmt.Printf("\nworst: %.4f%%   threshold: %.4f%%   missing/error: %d\n", worst, *threshold, missing)
	if worst > *threshold || missing > 0 {
		os.Exit(1)
	}
}

func indexByLabel(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".png" {
			continue
		}
		label := timestampPrefix.ReplaceAllString(name, "")
		out[label] = filepath.Join(dir, name)
	}
	return out, nil
}

func diffPNG(a, b string) (float64, error) {
	imgA, err := loadPNG(a)
	if err != nil {
		return 0, err
	}
	imgB, err := loadPNG(b)
	if err != nil {
		return 0, err
	}
	if imgA.Bounds() != imgB.Bounds() {
		return 100, fmt.Errorf("size mismatch %v vs %v", imgA.Bounds(), imgB.Bounds())
	}
	bounds := imgA.Bounds()
	total := bounds.Dx() * bounds.Dy()
	if total == 0 {
		return 0, nil
	}
	differ := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if imgA.At(x, y) != imgB.At(x, y) {
				differ++
			}
		}
	}
	return 100 * float64(differ) / float64(total), nil
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}
