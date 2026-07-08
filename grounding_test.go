package summarizer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileGroundingLoaderLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zta_definitions.json")
	want := "{\"signal\":\"definition\"}\n"
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	loader := NewFileGroundingLoader(path)
	got, err := loader.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got != want {
		t.Fatalf("Load() = %q, want %q", got, want)
	}
}
