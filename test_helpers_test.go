package summarizer

import (
	"encoding/json"
	"testing"
)

func validGuidanceForAnalysis(analysis *ReportAnalysis) *NarrativeGuidance {
	guidance := &NarrativeGuidance{
		Platforms: make([]PlatformGuidance, 0, len(analysis.Platforms)),
	}

	for _, platform := range analysis.Platforms {
		platformGuidance := PlatformGuidance{
			Name:                platform.Name,
			ZeroGroups:          []ZeroGroupGuidance{},
			Findings:            []ControlGuidance{},
			RemediationSequence: []string{},
			SharedBlockers:      []GuidanceBlocker{},
		}

		if platform.ZeroGuidanceMode == "individual" {
			for _, signal := range platform.ZeroSignals {
				platformGuidance.ZeroGroups = append(platformGuidance.ZeroGroups, ZeroGroupGuidance{
					Title:             signal.DisplayName,
					Signals:           []string{signal.Signal},
					TechnicalGuidance: testTechnicalGuidance(signal.DisplayName),
				})
			}
		} else if platform.ZeroGuidanceMode == "grouped" {
			signals := make([]string, 0, len(platform.ZeroSignals))
			for _, signal := range platform.ZeroSignals {
				signals = append(signals, signal.Signal)
			}
			platformGuidance.ZeroGroups = append(platformGuidance.ZeroGroups, ZeroGroupGuidance{
				Title:             "Shared prerequisite controls",
				Signals:           signals,
				TechnicalGuidance: testTechnicalGuidance("shared prerequisite controls"),
			})
		}

		for _, signal := range platform.PrioritySignals {
			platformGuidance.Findings = append(platformGuidance.Findings, ControlGuidance{
				Signal:            signal.Signal,
				TechnicalGuidance: testTechnicalGuidance(signal.DisplayName),
			})
		}
		if len(platform.ZeroSignals) > 0 || len(platform.PrioritySignals) > 0 {
			platformGuidance.RemediationSequence = []string{
				"Confirm shared prerequisites across the selected controls.",
				"Pilot approved changes and verify the resulting control state.",
			}
		}
		guidance.Platforms = append(guidance.Platforms, platformGuidance)
	}

	return guidance
}

func testTechnicalGuidance(control string) TechnicalGuidance {
	return TechnicalGuidance{
		WhyItMatters:         control + " defines an expected endpoint protection state; low coverage can leave systems without that protection.",
		SignalInterpretation: "The signal reflects the reported configured or active state, not only whether the capability exists.",
		RemediationSteps:     []string{"Confirm prerequisites for the platform policy.", "Apply the approved policy to a representative pilot group."},
		ChangeImpact:         ChangeImpact{Level: "Low", Rationale: "The policy is reversible and does not normally require downtime."},
		VerificationSteps:    []string{"Recheck the setting and confirm that it reports the intended enabled state."},
	}
}

func mustMarshalGuidance(t *testing.T, guidance *NarrativeGuidance) string {
	t.Helper()

	data, err := json.Marshal(guidance)
	if err != nil {
		t.Fatalf("marshal guidance: %v", err)
	}
	return string(data)
}
