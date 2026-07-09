package summarizer

import (
	"context"
	"testing"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestNewNarrativeGeneratorPlaceholder(t *testing.T) {
	cfg := &Config{NarrativeProvider: "placeholder"}

	generator, err := NewNarrativeGenerator(cfg)
	if err != nil {
		t.Fatalf("NewNarrativeGenerator returned error: %v", err)
	}

	report := &shared.CIDReport{
		CID:                 "cid-123",
		NumAIDs:             10,
		AverageOverallScore: 88.5,
		AverageOSScore:      90.0,
	}

	summary, err := generator.Summarize(context.Background(), report)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}
	if summary == "" {
		t.Fatal("expected placeholder summary to be non-empty")
	}
}

func TestNewNarrativeGeneratorRequiresProvider(t *testing.T) {
	cfg := &Config{}

	_, err := NewNarrativeGenerator(cfg)
	if err == nil {
		t.Fatal("expected missing narrative provider to return an error")
	}
}
