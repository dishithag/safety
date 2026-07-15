package summarizer

import (
	"strings"
	"testing"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestParseNarrativeGuidanceValidatesAndNormalizesResponse(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	guidance := validGuidanceForAnalysis(analysis)
	guidance.Platforms[0].Findings[0].RemediationDisruption.Level = "moderate"

	parsed, err := ParseNarrativeGuidance("```json\n"+mustMarshalGuidance(t, guidance)+"\n```", analysis)
	if err != nil {
		t.Fatalf("ParseNarrativeGuidance returned error: %v", err)
	}
	if got := parsed.Platforms[0].Findings[0].RemediationDisruption.Level; got != "Moderate" {
		t.Fatalf("normalized disruption level = %q, want Moderate", got)
	}
}

func TestParseNarrativeGuidanceRejectsContractViolations(t *testing.T) {
	analysis := testGuidanceAnalysis(t)

	tests := []struct {
		name   string
		mutate func(*NarrativeGuidance)
		want   string
	}{
		{
			name: "wrong priority signal",
			mutate: func(guidance *NarrativeGuidance) {
				guidance.Platforms[0].Findings[0].Signal = "invented_signal"
			},
			want: "finding 0 signal",
		},
		{
			name: "reordered individual zero groups",
			mutate: func(guidance *NarrativeGuidance) {
				groups := guidance.Platforms[0].ZeroGroups
				groups[0], groups[1] = groups[1], groups[0]
			},
			want: "individual group 0 signal",
		},
		{
			name: "unsupported disruption level",
			mutate: func(guidance *NarrativeGuidance) {
				guidance.Platforms[0].Findings[0].RemediationDisruption.Level = "Critical"
			},
			want: "want Low, Moderate, or High",
		},
		{
			name: "too few closing actions",
			mutate: func(guidance *NarrativeGuidance) {
				guidance.RecommendedNextSteps = []string{"Only one action."}
			},
			want: "want 3..5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guidance := validGuidanceForAnalysis(analysis)
			tt.mutate(guidance)
			_, err := ParseNarrativeGuidance(mustMarshalGuidance(t, guidance), analysis)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestParseNarrativeGuidanceRejectsUnknownFields(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	raw := strings.TrimSuffix(mustMarshalGuidance(t, validGuidanceForAnalysis(analysis)), "}") + `,"unexpected":true}`

	_, err := ParseNarrativeGuidance(raw, analysis)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v, want unknown field error", err)
	}
}

func testGuidanceAnalysis(t *testing.T) *ReportAnalysis {
	t.Helper()
	report := &shared.CIDReport{Platforms: []shared.PlatformSummary{{
		Name: "Windows 11",
		Compliance: shared.ComplianceMap{
			"zero_b": 0,
			"zero_a": 0,
			"low_a":  0.2,
			"low_b":  0.3,
			"low_c":  0.4,
		},
	}}}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	return analysis
}
