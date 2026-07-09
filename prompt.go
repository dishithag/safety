package summarizer

import (
	"encoding/json"
	"fmt"
	"strings"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

// BuildNarrativePrompt converts a CID report into the text sent to the configured LLM provider.
func BuildNarrativePrompt(report *shared.CIDReport) (string, error) {
	if report == nil {
		return "", fmt.Errorf("build narrative prompt: report is nil")
	}

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("build narrative prompt: marshal report: %w", err)
	}

	var b strings.Builder
	b.WriteString("You are writing a concise Zero Trust audit narrative for non-technical users.\n")
	b.WriteString("Return Markdown only.\n")
	b.WriteString("Use exactly these sections in this order:\n")
	b.WriteString("## Executive Summary\n")
	b.WriteString("## Top Findings\n")
	b.WriteString("## Per-Platform Highlights\n")
	b.WriteString("## Prioritized Recommendations\n\n")
	b.WriteString("Use only the supplied CID report JSON. Do not invent facts that are not present in the report.\n\n")
	b.WriteString("CID report JSON:\n```json\n")
	b.Write(reportJSON)
	b.WriteString("\n```\n")

	return b.String(), nil
}
