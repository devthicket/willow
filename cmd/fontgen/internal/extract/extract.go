// Package extract unpacks a .fontbundle archive into PNG and JSON files.
package extract

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Bundle unpacks all entries from a .fontbundle zip into outDir.
// If outDir is empty, the current directory is used.
func Bundle(data []byte, outDir string) error {
	if outDir == "" {
		var err error
		outDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("extract: open %s: %w", f.Name, err)
		}
		var buf bytes.Buffer
		_, err = buf.ReadFrom(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("extract: read %s: %w", f.Name, err)
		}

		dest := filepath.Join(outDir, filepath.Base(f.Name))
		if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("extract: write %s: %w", dest, err)
		}
		fmt.Printf("  %s\n", dest)
	}
	return nil
}
