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
	technical.MeasurementCaveat = "Confirm the exact evaluated state in the current ZTA documentation."
	technical.AdminTerminology = []string{"Memory integrity in Windows Security"}
	technical.Blockers = []GuidanceBlocker{{Blocker: "Legacy driver conflict", Response: "Update or replace the incompatible driver before enforcement."}}
	technical.FleetGuidance = "Start with compatible devices and isolate exceptions for review."

	markdown, err := RenderNarrativeMarkdown(analysis, guidance)
	if err != nil {
		t.Fatalf("RenderNarrativeMarkdown returned error: %v", err)
	}

	wants := []string{
		"# Zero Trust Assessment Report",
		"**CID:** `cid-render`",
		"**Reported devices:** **1,000**",
		"## High-Level Overview",
		"**61.25/100**",
		"Scores use a **0-100 scale**",
		"`average_overall_score`",
		"`average_os_score`",
		"`average_sensor_config_score`",
		"## Platform Analysis",
		"### 1. Windows 11",
		"#### Zero-Compliance Gaps",
		"does not by itself prove that a control is absent",
		"#### Priority Improvement Opportunities",
		"Cutoff ties are included.",
		"**What it is:**",
		"**Security impact:**",
		"**What ZTA evaluates:**",
		"**Measurement caveat:**",
		"**How to improve**",
		"**Remediation disruption:** **Low**",
		"**Operational considerations:**",
		"**Administrator terminology**",
		"**Common blockers**",
		"**Verification**",
		"**Fleet guidance:**",
		"## Recommended Next Steps",
		"> **Operational tip:**",
	}
	for _, want := range wants {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown missing %q\nfull markdown:\n%s", want, markdown)
		}
	}

	highLevel := strings.Split(markdown, "## Platform Analysis")[0]
	if strings.Contains(highLevel, "Windows 11") {
		t.Fatalf("high-level overview contains platform detail:\n%s", highLevel)
	}
	if strings.Contains(markdown, "## Executive Snapshot") {
		t.Fatalf("markdown contains an unwanted Executive Snapshot heading:\n%s", markdown)
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
