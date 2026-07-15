package summarizer

import (
	"context"
	"fmt"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

// NarrativeGenerator turns a CID report into a Markdown narrative.
type NarrativeGenerator interface {
	Summarize(ctx context.Context, report *shared.CIDReport) (string, error)
}

// NewNarrativeGenerator selects the configured narrative generation path.
func NewNarrativeGenerator(cfg *Config) (NarrativeGenerator, error) {
	switch effectiveNarrativeProvider(cfg.NarrativeProvider) {
	case "placeholder":
		return PlaceholderGenerator{}, nil
	case "genaihub":
		return NewGenAIHubGenerator(cfg)
	default:
		return nil, fmt.Errorf("unsupported narrative provider %q", cfg.NarrativeProvider)
	}
}

// PlaceholderGenerator keeps the local pipeline runnable without a real LLM.
type PlaceholderGenerator struct{}

// Summarize renders the deterministic placeholder Markdown used for offline work.
func (PlaceholderGenerator) Summarize(_ context.Context, report *shared.CIDReport) (string, error) {
	analysis, err := AnalyzeReport(report)
	if err != nil {
		return "", fmt.Errorf("summarize with placeholder: %w", err)
	}

	return RenderPlaceholderSummary(analysis)
}
