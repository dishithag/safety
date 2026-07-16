package summarizer

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfigBatchDefaults(t *testing.T) {
	t.Setenv("SUMMARIZER_CONCURRENCY", "")
	t.Setenv("SUMMARIZER_GENERATION_TIMEOUT", "")

	cfg, err := LoadConfig("test-service")
	if err != nil {
		t.Fatalf("LoadConfig returned unexpected error: %v", err)
	}
	if got, want := cfg.Concurrency, 3; got != want {
		t.Fatalf("Concurrency = %d, want %d", got, want)
	}
	if got, want := cfg.GenerationTimeout, 5*time.Minute; got != want {
		t.Fatalf("GenerationTimeout = %s, want %s", got, want)
	}
}

func TestLoadConfigBatchOverrides(t *testing.T) {
	t.Setenv("SUMMARIZER_CONCURRENCY", "7")
	t.Setenv("SUMMARIZER_GENERATION_TIMEOUT", "90s")

	cfg, err := LoadConfig("test-service")
	if err != nil {
		t.Fatalf("LoadConfig returned unexpected error: %v", err)
	}
	if got, want := cfg.Concurrency, 7; got != want {
		t.Fatalf("Concurrency = %d, want %d", got, want)
	}
	if got, want := cfg.GenerationTimeout, 90*time.Second; got != want {
		t.Fatalf("GenerationTimeout = %s, want %s", got, want)
	}
}

func TestLoadConfigRejectsInvalidBatchSettings(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "non-numeric concurrency", key: "SUMMARIZER_CONCURRENCY", value: "many"},
		{name: "zero concurrency", key: "SUMMARIZER_CONCURRENCY", value: "0"},
		{name: "invalid timeout", key: "SUMMARIZER_GENERATION_TIMEOUT", value: "later"},
		{name: "zero timeout", key: "SUMMARIZER_GENERATION_TIMEOUT", value: "0s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SUMMARIZER_CONCURRENCY", "")
			t.Setenv("SUMMARIZER_GENERATION_TIMEOUT", "")
			t.Setenv(test.key, test.value)

			_, err := LoadConfig("test-service")
			if err == nil {
				t.Fatal("LoadConfig returned nil error")
			}
			if !strings.Contains(err.Error(), test.key) {
				t.Fatalf("error = %q, want it to mention %s", err, test.key)
			}
		})
	}
}
