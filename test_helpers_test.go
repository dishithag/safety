package summarizer

import (
	"encoding/json"
	"testing"
)

func validGuidanceForAnalysis(analysis *ReportAnalysis) *NarrativeGuidance {
	guidance := &NarrativeGuidance{
		Platforms:            make([]PlatformGuidance, 0, len(analysis.Platforms)),
		RecommendedNextSteps: []string{"Review zero-compliance gaps.", "Pilot approved changes.", "Verify the next assessment."},
		OperationalTip:       "Use a representative pilot group before broad deployment.",
	}

	for _, platform := range analysis.Platforms {
		platformGuidance := PlatformGuidance{
			Name:       platform.Name,
			ZeroGroups: []ZeroGroupGuidance{},
			Findings:   []ControlGuidance{},
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
		guidance.Platforms = append(guidance.Platforms, platformGuidance)
	}

	return guidance
}

func testTechnicalGuidance(control string) TechnicalGuidance {
	return TechnicalGuidance{
		Meaning:                   control + " defines the expected endpoint protection state.",
		SecurityImpact:            "Low coverage can leave endpoints without the intended protection.",
		ZTAEvaluation:             "ZTA evaluates the reported configured or active state, not only whether the capability exists.",
		RemediationSteps:          []string{"Confirm prerequisites for the platform policy.", "Apply the approved policy to a representative pilot group."},
		RemediationDisruption:     RemediationDisruption{Level: "Low", Rationale: "The policy is reversible and does not normally require downtime."},
		OperationalConsiderations: "Test policy compatibility and retain a rollback path.",
		VerificationSteps:         []string{"Recheck the setting and confirm that it reports the intended enabled state."},
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
