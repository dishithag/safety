package summarizer

import (
	"strings"
	"testing"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestRenderNarrativeMarkdownUsesStableActionableStructure(t *testing.T) {
	report := &shared.CIDReport{
		CID:                      "cid-render",
		NumAIDs:                  1000,
		AverageOverallScore:      61.25,
		AverageOSScore:           55.5,
		AverageSensorConfigScore: 67,
		Platforms: []shared.PlatformSummary{
			{
				Name:                     "Windows 11",
				NumAIDs:                  1000,
				AverageOverallScore:      61.25,
				AverageOSScore:           55.5,
				AverageSensorConfigScore: 67,
				Compliance: shared.ComplianceMap{
					"secure_boot_enabled": 0,
					"hvci_enabled":        0.2,
					"iommu_in_use":        0.3,
					"smm_protections":     0.4,
					"full_control":        1,
				},
			},
		},
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	guidance := validGuidanceForAnalysis(analysis)
	technical := &guidance.Platforms[0].Findings[0].TechnicalGuidance
	technical.AdminTerminology = []string{"Memory integrity in Windows Security"}
	guidance.Platforms[0].SharedBlockers = []GuidanceBlocker{{Blocker: "Legacy driver conflict", Response: "Update or replace the incompatible driver before enforcement."}}
	guidance.Platforms[0].FleetGuidance = "Start with compatible devices and isolate exceptions for review."

	markdown, err := RenderNarrativeMarkdown(analysis, guidance)
	if err != nil {
		t.Fatalf("RenderNarrativeMarkdown returned error: %v", err)
	}

	wants := []string{
		"# Zero Trust Assessment Report",
		"**CID:** `cid-render`",
		"**Reported devices:** **1,000**",
		"## High-Level Overview",
		"**Overall posture: 61.25/100.**",
		"Scores use a **0-100 scale**",
		"## 1. Windows 11",
		"### Zero-Compliance Gaps",
		"> ZTA recorded **0% compliance**",
		"does not by itself prove that a control is absent",
		"| Control | Signal ID | Reported Compliance |",
		"\n---\n\n### Lowest-Coverage Improvement Opportunities",
		"This is a coverage-based selection, not a security-severity ranking.",
		"| Control | Signal ID | Compliance |",
		"- **Why it matters:**",
		"- **Signal interpretation:**",
		"- **Also called:**",
		"##### Remediation",
		"##### Change Impact",
		"- **Estimated change impact:** **Low**",
		"##### Verification",
		"### Recommended Remediation Sequence",
		"#### Shared Blockers",
		"#### Fleet Rollout Guidance",
		"## Recommended Next Steps",
	}
	for _, want := range wants {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown missing %q\nfull markdown:\n%s", want, markdown)
		}
	}

	highLevel := strings.Split(markdown, "## 1. Windows 11")[0]
	if strings.Contains(highLevel, "Windows 11") {
		t.Fatalf("high-level overview contains platform detail:\n%s", highLevel)
	}
	for _, unwanted := range []string{
		"Cutoff ties",
		"cutoff ties",
		"average_overall_score",
		"average_os_score",
		"average_sensor_config_score",
		"Operational tip",
	} {
		if strings.Contains(markdown, unwanted) {
			t.Fatalf("markdown contains internal or redundant text %q:\n%s", unwanted, markdown)
		}
	}
	if got := strings.Count(markdown, "**Reported devices:**"); got != 1 {
		t.Fatalf("reported device label appears %d times, want once:\n%s", got, markdown)
	}
	for _, score := range []string{"61.25/100", "55.50/100", "67.00/100"} {
		if got := strings.Count(markdown, score); got != 1 {
			t.Fatalf("score %q appears %d times, want once:\n%s", score, got, markdown)
		}
	}
	for _, signalID := range []string{"secure_boot_enabled", "hvci_enabled", "iommu_in_use", "smm_protections"} {
		if got := strings.Count(markdown, "`"+signalID+"`"); got != 1 {
			t.Fatalf("signal ID %q appears %d times, want once in its summary table:\n%s", signalID, got, markdown)
		}
	}
	if strings.Contains(markdown, "## Executive Snapshot") {
		t.Fatalf("markdown contains an unwanted Executive Snapshot heading:\n%s", markdown)
	}
}

func TestRenderFallbackSummaryKeepsCurrentFactsAndExplainsDegradedGuidance(t *testing.T) {
	report := &shared.CIDReport{
		CID:                 "cid-fallback",
		NumAIDs:             12,
		AverageOverallScore: 42,
		AverageOSScore:      38,
		Platforms: []shared.PlatformSummary{{
			Name:                "Windows 11",
			NumAIDs:             12,
			AverageOverallScore: 42,
			AverageOSScore:      38,
			Compliance: shared.ComplianceMap{
				"secure_boot_enabled": 0,
				"hvci_enabled":        0.25,
			},
		}},
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}

	markdown, err := RenderFallbackSummary(analysis)
	if err != nil {
		t.Fatalf("RenderFallbackSummary returned error: %v", err)
	}
	for _, want := range []string{
		"**Guidance status:** Detailed generated remediation was unavailable",
		"**Overall posture: 42.00/100.**",
		"`secure_boot_enabled`",
		"`hvci_enabled`",
		"### Zero-Compliance Gaps",
		"### Lowest-Coverage Improvement Opportunities",
		"## Recommended Next Steps",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("fallback markdown missing %q\nfull markdown:\n%s", want, markdown)
		}
	}
	for _, unwanted := range []string{
		"NARRATIVE_PROVIDER=placeholder",
		"##### Remediation",
		"Detailed remediation guidance is unavailable",
	} {
		if strings.Contains(markdown, unwanted) {
			t.Fatalf("fallback markdown contains unwanted text %q:\n%s", unwanted, markdown)
		}
	}
}

func TestRenderPlaceholderSummaryKeepsDeterministicReportShell(t *testing.T) {
	report := &shared.CIDReport{
		CID:                 "cid-placeholder",
		NumAIDs:             5,
		AverageOverallScore: 100,
		AverageOSScore:      100,
		Platforms: []shared.PlatformSummary{{
			Name:                "iOS",
			NumAIDs:             5,
			AverageOverallScore: 100,
			AverageOSScore:      100,
			Compliance:          shared.ComplianceMap{"lock_screen_enabled_ios": 1},
		}},
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}

	markdown, err := RenderPlaceholderSummary(analysis)
	if err != nil {
		t.Fatalf("RenderPlaceholderSummary returned error: %v", err)
	}
	for _, want := range []string{
		"# Zero Trust Assessment Report",
		"**Positive observation:** All **1 tracked signals** report **100% compliance**.",
		"## 1. iOS",
		"No remediation findings were selected",
		"## Recommended Next Steps",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("placeholder markdown missing %q\nfull markdown:\n%s", want, markdown)
		}
	}
	if strings.Contains(markdown, "Technical remediation guidance is omitted") {
		t.Fatalf("fully compliant platform should not include omitted-remediation message:\n%s", markdown)
	}
}
