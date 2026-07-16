package summarizer

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const maxGroupedZeroGuidance = 5

// NarrativeGuidance is the bounded, structured content returned by the LLM.
type NarrativeGuidance struct {
	Platforms            []PlatformGuidance `json:"platforms"`
	RecommendedNextSteps []string           `json:"recommended_next_steps"`
}

// PlatformGuidance contains technical guidance for one analyzed platform.
type PlatformGuidance struct {
	Name                string              `json:"name"`
	ZeroGroups          []ZeroGroupGuidance `json:"zero_groups"`
	Findings            []ControlGuidance   `json:"findings"`
	RemediationSequence []string            `json:"remediation_sequence"`
	SharedBlockers      []GuidanceBlocker   `json:"shared_blockers"`
	FleetGuidance       string              `json:"fleet_guidance"`
}

// ZeroGroupGuidance provides individual or grouped guidance for zero-compliance signals.
type ZeroGroupGuidance struct {
	Title   string   `json:"title"`
	Signals []string `json:"signals"`
	TechnicalGuidance
}

// ControlGuidance provides technical guidance for one selected partial-compliance signal.
type ControlGuidance struct {
	Signal string `json:"signal"`
	TechnicalGuidance
}

// TechnicalGuidance captures the required and optional content for an actionable finding.
type TechnicalGuidance struct {
	Meaning                   string                `json:"meaning"`
	SecurityImpact            string                `json:"security_impact"`
	ZTAEvaluation             string                `json:"zta_evaluation"`
	MeasurementCaveat         string                `json:"measurement_caveat,omitempty"`
	RemediationSteps          []string              `json:"remediation_steps"`
	RemediationDisruption     RemediationDisruption `json:"remediation_disruption"`
	OperationalConsiderations string                `json:"operational_considerations"`
	VerificationSteps         []string              `json:"verification_steps"`
	AdminTerminology          []string              `json:"admin_terminology,omitempty"`
}

// RemediationDisruption describes the operational risk of making a change.
type RemediationDisruption struct {
	Level     string `json:"level"`
	Rationale string `json:"rationale"`
}

// GuidanceBlocker maps a common implementation blocker to its response.
type GuidanceBlocker struct {
	Blocker  string `json:"blocker"`
	Response string `json:"response"`
}

// ParseNarrativeGuidance decodes and validates a model response against the prepared report facts.
func ParseNarrativeGuidance(raw string, analysis *ReportAnalysis) (*NarrativeGuidance, error) {
	if analysis == nil {
		return nil, fmt.Errorf("parse narrative guidance: analysis is nil")
	}

	response, err := unwrapJSONResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse narrative guidance: %w", err)
	}

	decoder := json.NewDecoder(strings.NewReader(response))
	decoder.DisallowUnknownFields()
	var guidance NarrativeGuidance
	if err := decoder.Decode(&guidance); err != nil {
		return nil, fmt.Errorf("parse narrative guidance JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, fmt.Errorf("parse narrative guidance JSON: %w", err)
	}
	if err := validateNarrativeGuidance(&guidance, analysis); err != nil {
		return nil, fmt.Errorf("validate narrative guidance: %w", err)
	}
	return &guidance, nil
}

func unwrapJSONResponse(raw string) (string, error) {
	response := strings.TrimSpace(raw)
	if response == "" {
		return "", fmt.Errorf("response is empty")
	}
	if !strings.HasPrefix(response, "```") {
		return response, nil
	}

	lines := strings.Split(response, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[len(lines)-1]) != "```" {
		return "", fmt.Errorf("response contains an incomplete JSON code fence; model output may have been truncated")
	}
	openingFence := strings.TrimSpace(lines[0])
	language := strings.TrimSpace(strings.TrimPrefix(openingFence, "```"))
	if !strings.HasPrefix(openingFence, "```") || (language != "" && !strings.EqualFold(language, "json")) {
		return "", fmt.Errorf("response contains an unsupported JSON code fence %q", openingFence)
	}
	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n")), nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("response contains more than one JSON value")
		}
		return err
	}
	return nil
}

func validateNarrativeGuidance(guidance *NarrativeGuidance, analysis *ReportAnalysis) error {
	if len(guidance.Platforms) != len(analysis.Platforms) {
		return fmt.Errorf("platform count = %d, want %d", len(guidance.Platforms), len(analysis.Platforms))
	}

	for i := range analysis.Platforms {
		expected := &analysis.Platforms[i]
		actual := &guidance.Platforms[i]
		if actual.Name != expected.Name {
			return fmt.Errorf("platform %d name = %q, want %q", i, actual.Name, expected.Name)
		}
		if err := validateZeroGroups(actual.ZeroGroups, expected); err != nil {
			return fmt.Errorf("platform %q zero guidance: %w", expected.Name, err)
		}
		if len(actual.Findings) != len(expected.PrioritySignals) {
			return fmt.Errorf("platform %q finding count = %d, want %d", expected.Name, len(actual.Findings), len(expected.PrioritySignals))
		}
		for findingIndex := range expected.PrioritySignals {
			finding := &actual.Findings[findingIndex]
			expectedSignal := expected.PrioritySignals[findingIndex].Signal
			if finding.Signal != expectedSignal {
				return fmt.Errorf("platform %q finding %d signal = %q, want %q", expected.Name, findingIndex, finding.Signal, expectedSignal)
			}
			if err := validateTechnicalGuidance(&finding.TechnicalGuidance); err != nil {
				return fmt.Errorf("platform %q signal %q: %w", expected.Name, finding.Signal, err)
			}
		}
		if err := validatePlatformImplementation(actual, expected); err != nil {
			return fmt.Errorf("platform %q implementation guidance: %w", expected.Name, err)
		}
	}

	if len(guidance.RecommendedNextSteps) < 3 || len(guidance.RecommendedNextSteps) > 5 {
		return fmt.Errorf("recommended_next_steps count = %d, want 3..5", len(guidance.RecommendedNextSteps))
	}
	for i, step := range guidance.RecommendedNextSteps {
		if strings.TrimSpace(step) == "" {
			return fmt.Errorf("recommended_next_steps[%d] is empty", i)
		}
	}
	return nil
}

func validatePlatformImplementation(guidance *PlatformGuidance, platform *PlatformAnalysis) error {
	hasFindings := len(platform.ZeroSignals) > 0 || len(platform.PrioritySignals) > 0
	if !hasFindings {
		if len(guidance.RemediationSequence) != 0 || len(guidance.SharedBlockers) != 0 || strings.TrimSpace(guidance.FleetGuidance) != "" {
			return fmt.Errorf("received implementation guidance for a platform without findings")
		}
		return nil
	}

	if len(guidance.RemediationSequence) < 2 || len(guidance.RemediationSequence) > 5 {
		return fmt.Errorf("remediation_sequence count = %d, want 2..5", len(guidance.RemediationSequence))
	}
	for i, step := range guidance.RemediationSequence {
		if strings.TrimSpace(step) == "" {
			return fmt.Errorf("remediation_sequence[%d] is empty", i)
		}
	}
	if len(guidance.SharedBlockers) > 3 {
		return fmt.Errorf("shared_blockers count = %d, maximum is 3", len(guidance.SharedBlockers))
	}
	for i, blocker := range guidance.SharedBlockers {
		if strings.TrimSpace(blocker.Blocker) == "" || strings.TrimSpace(blocker.Response) == "" {
			return fmt.Errorf("shared_blockers[%d] requires blocker and response", i)
		}
	}
	return nil
}

func validateZeroGroups(groups []ZeroGroupGuidance, platform *PlatformAnalysis) error {
	if len(platform.ZeroSignals) == 0 {
		if len(groups) != 0 {
			return fmt.Errorf("received %d groups for a platform without zero signals", len(groups))
		}
		return nil
	}
	if len(groups) == 0 {
		return fmt.Errorf("zero groups are missing")
	}

	expected := make(map[string]struct{}, len(platform.ZeroSignals))
	for _, signal := range platform.ZeroSignals {
		expected[signal.Signal] = struct{}{}
	}
	seen := make(map[string]struct{}, len(platform.ZeroSignals))

	if platform.ZeroGuidanceMode == "individual" && len(groups) != len(platform.ZeroSignals) {
		return fmt.Errorf("individual group count = %d, want %d", len(groups), len(platform.ZeroSignals))
	}
	if platform.ZeroGuidanceMode == "grouped" && len(groups) > maxGroupedZeroGuidance {
		return fmt.Errorf("grouped zero guidance count = %d, maximum is %d", len(groups), maxGroupedZeroGuidance)
	}

	for i := range groups {
		group := &groups[i]
		if strings.TrimSpace(group.Title) == "" {
			return fmt.Errorf("group %d title is empty", i)
		}
		if len(group.Signals) == 0 {
			return fmt.Errorf("group %q has no signals", group.Title)
		}
		if platform.ZeroGuidanceMode == "individual" && len(group.Signals) != 1 {
			return fmt.Errorf("individual group %q contains %d signals, want 1", group.Title, len(group.Signals))
		}
		if platform.ZeroGuidanceMode == "individual" && group.Signals[0] != platform.ZeroSignals[i].Signal {
			return fmt.Errorf("individual group %d signal = %q, want %q", i, group.Signals[0], platform.ZeroSignals[i].Signal)
		}
		for _, signal := range group.Signals {
			if _, ok := expected[signal]; !ok {
				return fmt.Errorf("group %q contains unexpected signal %q", group.Title, signal)
			}
			if _, duplicate := seen[signal]; duplicate {
				return fmt.Errorf("signal %q appears in more than one zero group", signal)
			}
			seen[signal] = struct{}{}
		}
		if err := validateTechnicalGuidance(&group.TechnicalGuidance); err != nil {
			return fmt.Errorf("group %q: %w", group.Title, err)
		}
	}

	if len(seen) != len(expected) {
		var missing []string
		for signal := range expected {
			if _, ok := seen[signal]; !ok {
				missing = append(missing, signal)
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("zero signals missing from guidance: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateTechnicalGuidance(guidance *TechnicalGuidance) error {
	required := []struct {
		name  string
		value string
	}{
		{"meaning", guidance.Meaning},
		{"security_impact", guidance.SecurityImpact},
		{"zta_evaluation", guidance.ZTAEvaluation},
		{"remediation_disruption.rationale", guidance.RemediationDisruption.Rationale},
		{"operational_considerations", guidance.OperationalConsiderations},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is empty", field.name)
		}
	}

	level := strings.ToLower(strings.TrimSpace(guidance.RemediationDisruption.Level))
	switch level {
	case "low":
		guidance.RemediationDisruption.Level = "Low"
	case "moderate":
		guidance.RemediationDisruption.Level = "Moderate"
	case "high":
		guidance.RemediationDisruption.Level = "High"
	default:
		return fmt.Errorf("remediation_disruption.level = %q, want Low, Moderate, or High", guidance.RemediationDisruption.Level)
	}

	if len(guidance.RemediationSteps) < 2 || len(guidance.RemediationSteps) > 4 {
		return fmt.Errorf("remediation_steps count = %d, want 2..4", len(guidance.RemediationSteps))
	}
	for i, step := range guidance.RemediationSteps {
		if strings.TrimSpace(step) == "" {
			return fmt.Errorf("remediation_steps[%d] is empty", i)
		}
	}
	if len(guidance.VerificationSteps) == 0 || len(guidance.VerificationSteps) > 2 {
		return fmt.Errorf("verification_steps count = %d, want 1..2", len(guidance.VerificationSteps))
	}
	for i, step := range guidance.VerificationSteps {
		if strings.TrimSpace(step) == "" {
			return fmt.Errorf("verification_steps[%d] is empty", i)
		}
	}
	if len(guidance.AdminTerminology) > 2 {
		return fmt.Errorf("admin_terminology count = %d, maximum is 2", len(guidance.AdminTerminology))
	}
	return nil
}
