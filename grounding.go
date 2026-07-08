package summarizer

import (
	"fmt"
	"os"
	"strings"
)

// GroundingLoader loads the signal-grounding JSON used in LLM prompts.
type GroundingLoader interface {
	Load() (string, error)
}

// GroundingLoaderFunc adapts a function into a GroundingLoader.
type GroundingLoaderFunc func() (string, error)

// Load implements GroundingLoader.
func (f GroundingLoaderFunc) Load() (string, error) {
	return f()
}

// FileGroundingLoader reads grounding JSON from disk.
type FileGroundingLoader struct {
	path string
}

// NewFileGroundingLoader creates a file-backed grounding loader.
func NewFileGroundingLoader(path string) *FileGroundingLoader {
	return &FileGroundingLoader{path: path}
}

// Load returns the raw grounding JSON contents.
func (l *FileGroundingLoader) Load() (string, error) {
	if strings.TrimSpace(l.path) == "" {
		return "", fmt.Errorf("load grounding: ZTA_DEFINITIONS_PATH is not set")
	}

	body, err := os.ReadFile(l.path)
	if err != nil {
		return "", fmt.Errorf("load grounding: read %s: %w", l.path, err)
	}

	return string(body), nil
}
