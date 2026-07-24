package summarizer

import (
	"errors"
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
		Platforms: []shared.PlatformSummary{{
			Name:                "Windows 11",
			NumAIDs:             42,
			AverageOverallScore: 91.2,
			AverageOSScore:      89.4,
			Compliance: shared.ComplianceMap{
				"secure_boot_enabled": 0,
				"hvci_enabled":        0.2,
				"iommu_in_use":        0.3,
				"smm_protections":     0.4,
			},
		}},
	}

	prompt, err := BuildNarrativePrompt(report)
	if err != nil {
		t.Fatalf("BuildNarrativePrompt returned error: %v", err)
	}

	wants := []string{
		"Return exactly one JSON object",
		"Do not return Markdown",
		"application, not you, owns report selection",
		"Do not claim an exact ZTA scoring formula",
		"copy each platform name exactly",
		"Do not describe the internal selection method",
		"do not return display_name",
		"zero to three useful vendor-facing names",
		"treat zero_groups as a partition",
		"choose its primary remediation path",
		"Never duplicate the signal across groups",
		"FINAL SELF-CHECK BEFORE RETURNING JSON",
		`"why_it_matters"`,
		`"signal_interpretation"`,
		`"change_impact"`,
		"PLATFORM-LEVEL IMPLEMENTATION RULES",
		`"remediation_sequence"`,
		`"shared_blockers"`,
		`"zero_guidance_mode": "individual"`,
		`"signal": "secure_boot_enabled"`,
		`"signal": "hvci_enabled"`,
		`"signal": "iommu_in_use"`,
		`"signal": "smm_protections"`,
		"Windows 11",
		report.CID,
	}
	for _, want := range wants {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\nfull prompt:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "0%% compliance") {
		t.Fatalf("prompt contains an escaped display artifact:\n%s", prompt)
	}
	if strings.Contains(prompt, `"operational_tip"`) {
		t.Fatalf("prompt contains removed operational_tip field:\n%s", prompt)
	}
	for _, removed := range []string{`"recommended_next_steps"`, `"zta_evaluation"`, `"measurement_caveat"`} {
		if strings.Contains(prompt, removed) {
			t.Fatalf("prompt contains removed field %q:\n%s", removed, prompt)
		}
	}
}

func TestBuildGuidanceRepairPromptIncludesFailureAsData(t *testing.T) {
	analysis := &ReportAnalysis{CID: "cid-repair"}
	repairPrompt, err := BuildGuidanceRepairPrompt(
		analysis,
		`{"platforms":[],"unexpected":"value"}`,
		errors.New("wrong platform count"),
	)
	if err != nil {
		t.Fatalf("BuildGuidanceRepairPrompt returned error: %v", err)
	}
	for _, want := range []string{
		"REPAIR TASK",
		"Validation error: wrong platform count",
		`"{\"platforms\":[],\"unexpected\":\"value\"}"`,
		"Return only the corrected full JSON object",
	} {
		if !strings.Contains(repairPrompt, want) {
			t.Fatalf("repair prompt missing %q:\n%s", want, repairPrompt)
		}
	}
}
