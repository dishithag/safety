package summarizer

import (
	"encoding/json"
	"fmt"
	"strings"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

// BuildNarrativePrompt converts a CID report plus grounding JSON into the text
// sent to the configured LLM provider.
func BuildNarrativePrompt(report *shared.CIDReport, grounding string) (string, error) {
	if report == nil {
		return "", fmt.Errorf("build narrative prompt: report is nil")
	}
	if strings.TrimSpace(grounding) == "" {
		return "", fmt.Errorf("build narrative prompt: grounding JSON is empty")
	}

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("build narrative prompt: marshal report: %w", err)
	}

	var b strings.Builder
	b.WriteString("You are writing a Zero Trust audit narrative for one customer.\n")
	b.WriteString("Return Markdown only.\n")
	b.WriteString("Use exactly these sections in this order:\n")
	b.WriteString("## Executive Summary\n")
	b.WriteString("## Top Findings\n")
	b.WriteString("## Per-Platform Highlights\n")
	b.WriteString("## Prioritized Recommendations\n\n")
	b.WriteString("Every claim must be grounded in the supplied report JSON and signal definitions.\n\n")
	b.WriteString("Signal definitions JSON:\n```json\n")
	b.WriteString(strings.TrimSpace(grounding))
	b.WriteString("\n```\n\n")
	b.WriteString("CID report JSON:\n```json\n")
	b.Write(reportJSON)
	b.WriteString("\n```\n")

	return b.String(), nil
}
