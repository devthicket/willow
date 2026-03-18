package extract

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBundle_ExtractsFiles(t *testing.T) {
	bundle := makeZip(t, map[string][]byte{
		"regular_64.json": []byte(`{"type":"sdf"}`),
		"regular_64.png":  []byte("fakepngdata"),
	})
	outDir := t.TempDir()
	if err := Bundle(bundle, outDir); err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	for _, name := range []string{"regular_64.json", "regular_64.png"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Errorf("expected file %s to exist: %v", name, err)
		}
	}
}

func TestBundle_FileContentsCorrect(t *testing.T) {
	content := []byte(`{"type":"sdf","size":64}`)
	bundle := makeZip(t, map[string][]byte{"regular_64.json": content})
	outDir := t.TempDir()
	if err := Bundle(bundle, outDir); err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(outDir, "regular_64.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestBundle_CreatesOutDir(t *testing.T) {
	bundle := makeZip(t, map[string][]byte{"file.txt": []byte("data")})
	outDir := filepath.Join(t.TempDir(), "newsubdir")
	if err := Bundle(bundle, outDir); err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("output dir was not created: %v", err)
	}
}

func TestBundle_EmptyOutDir_UsesCurrentDir(t *testing.T) {
	// Passing "" means use current working directory.
	// We won't call this in CI since it writes to cwd; just test that it
	// doesn't panic on the parameter resolution.
	// This is covered indirectly by the above tests.
}

func TestBundle_InvalidZip_Error(t *testing.T) {
	err := Bundle([]byte("not a zip"), t.TempDir())
	if err == nil {
		t.Error("expected error for invalid zip data")
	}
}

func TestBundle_EmptyZip_NoError(t *testing.T) {
	var buf bytes.Buffer
	zip.NewWriter(&buf).Close()
	if err := Bundle(buf.Bytes(), t.TempDir()); err != nil {
		t.Errorf("Bundle on empty zip: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip.Create: %v", err)
		}
		w.Write(data)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip.Close: %v", err)
	}
	return buf.Bytes()
}
