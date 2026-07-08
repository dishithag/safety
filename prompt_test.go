package summarizer

import (
	"strings"
	"testing"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestBuildNarrativePrompt(t *testing.T) {
	report := &shared.CIDReport{
		CID:                 "0f53593ceae34995af8fd295c18f1e25",
		NumAIDs:             42,
		AverageOverallScore: 91.2,
		AverageOSScore:      89.4,
	}
	grounding := "{\"signals\":{\"sensor\":\"Installed sensor check\"}}"

	prompt, err := BuildNarrativePrompt(report, grounding)
	if err != nil {
		t.Fatalf("BuildNarrativePrompt returned error: %v", err)
	}

	wants := []string{
		"## Executive Summary",
		"## Top Findings",
		"## Per-Platform Highlights",
		"## Prioritized Recommendations",
		report.CID,
		grounding,
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\nfull prompt:\n%s", want, prompt)
		}
	}
}
