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
		"application owns all presentation",
		"Do not claim an exact ZTA scoring formula",
		"use each platform name exactly",
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
}
