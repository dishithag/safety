package summarizer

import (
	"reflect"
	"strings"
	"testing"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestParseNarrativeGuidanceValidatesAndNormalizesResponse(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	guidance := validGuidanceForAnalysis(analysis)
	guidance.Platforms[0].Findings[0].ChangeImpact.Level = "moderate"

	parsed, err := ParseNarrativeGuidance("```json\n"+mustMarshalGuidance(t, guidance)+"\n```", analysis)
	if err != nil {
		t.Fatalf("ParseNarrativeGuidance returned error: %v", err)
	}
	if got := parsed.Platforms[0].Findings[0].ChangeImpact.Level; got != "Moderate" {
		t.Fatalf("normalized disruption level = %q, want Moderate", got)
	}
}

func TestParseNarrativeGuidanceAcceptsJSONFenceVariants(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	raw := "``` JSON\n" + mustMarshalGuidance(t, validGuidanceForAnalysis(analysis)) + "\n```"

	if _, err := ParseNarrativeGuidance(raw, analysis); err != nil {
		t.Fatalf("ParseNarrativeGuidance rejected a valid JSON fence variant: %v", err)
	}
}

func TestParseNarrativeGuidanceAcceptsCompatibilityDisplayNames(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	raw := mustMarshalGuidance(t, validGuidanceForAnalysis(analysis))
	raw = strings.Replace(raw, `"signals":`, `"display_name":"Zero A","signals":`, 1)
	raw = strings.Replace(raw, `"signal":"low_a"`, `"signal":"low_a","display_name":"Low A"`, 1)

	if _, err := ParseNarrativeGuidance(raw, analysis); err != nil {
		t.Fatalf("ParseNarrativeGuidance rejected compatibility display_name fields: %v", err)
	}
}

func TestParseNarrativeGuidancePreservesAdministratorTerminology(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	guidance := validGuidanceForAnalysis(analysis)
	want := []string{"Memory integrity", "HVCI", "Virtualization-based security"}
	guidance.Platforms[0].Findings[0].AdminTerminology = want

	parsed, err := ParseNarrativeGuidance(mustMarshalGuidance(t, guidance), analysis)
	if err != nil {
		t.Fatalf("ParseNarrativeGuidance rejected useful administrator terminology: %v", err)
	}
	got := parsed.Platforms[0].Findings[0].AdminTerminology
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AdminTerminology = %v, want %v", got, want)
	}
}

func TestParseNarrativeGuidanceNormalizesExcessAdministratorTerminology(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	guidance := validGuidanceForAnalysis(analysis)
	guidance.Platforms[0].Findings[0].AdminTerminology = []string{
		" Memory integrity ",
		"HVCI",
		"hvci",
		"Virtualization-based security",
		"Device Guard",
	}

	parsed, err := ParseNarrativeGuidance(mustMarshalGuidance(t, guidance), analysis)
	if err != nil {
		t.Fatalf("ParseNarrativeGuidance rejected excess administrator terminology: %v", err)
	}
	want := []string{"Memory integrity", "HVCI", "Virtualization-based security"}
	got := parsed.Platforms[0].Findings[0].AdminTerminology
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AdminTerminology = %v, want normalized values %v", got, want)
	}
}

func TestParseNarrativeGuidanceReportsIncompleteFence(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	raw := "```json\n" + mustMarshalGuidance(t, validGuidanceForAnalysis(analysis))

	_, err := ParseNarrativeGuidance(raw, analysis)
	if err == nil || !strings.Contains(err.Error(), "may have been truncated") {
		t.Fatalf("error = %v, want possible truncation context", err)
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
				guidance.Platforms[0].Findings[0].ChangeImpact.Level = "Critical"
			},
			want: "want Low, Moderate, or High",
		},
		{
			name: "missing platform remediation sequence",
			mutate: func(guidance *NarrativeGuidance) {
				guidance.Platforms[0].RemediationSequence = nil
			},
			want: "remediation_sequence count",
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

func TestParseNarrativeGuidanceNormalizesDuplicateGroupedZeroSignals(t *testing.T) {
	report := &shared.CIDReport{Platforms: []shared.PlatformSummary{{
		Name: "Windows 11",
		Compliance: shared.ComplianceMap{
			"zero_a": 0,
			"zero_b": 0,
			"zero_c": 0,
			"zero_d": 0,
			"zero_e": 0,
			"zero_f": 0,
		},
	}}}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	guidance := validGuidanceForAnalysis(analysis)
	signals := analysis.Platforms[0].ZeroSignals
	guidance.Platforms[0].ZeroGroups = []ZeroGroupGuidance{
		{
			Title:             "First theme",
			Signals:           []string{signals[0].Signal, signals[1].Signal, signals[2].Signal},
			TechnicalGuidance: testTechnicalGuidance("first theme"),
		},
		{
			Title:             "Second theme",
			Signals:           []string{signals[0].Signal, signals[3].Signal, signals[4].Signal, signals[5].Signal},
			TechnicalGuidance: testTechnicalGuidance("second theme"),
		},
	}

	parsed, err := ParseNarrativeGuidance(mustMarshalGuidance(t, guidance), analysis)
	if err != nil {
		t.Fatalf("ParseNarrativeGuidance rejected a repairable duplicate assignment: %v", err)
	}
	if got := parsed.Platforms[0].ZeroGroups[1].Signals; reflect.DeepEqual(got, guidance.Platforms[0].ZeroGroups[1].Signals) {
		t.Fatalf("duplicate grouped signal was not removed: %v", got)
	}
	for _, signal := range parsed.Platforms[0].ZeroGroups[1].Signals {
		if signal == signals[0].Signal {
			t.Fatalf("duplicate signal %q remains in the second group", signal)
		}
	}
}

func TestParseNarrativeGuidanceStillRejectsUnknownGroupedZeroSignals(t *testing.T) {
	report := &shared.CIDReport{Platforms: []shared.PlatformSummary{{
		Name: "Windows 11",
		Compliance: shared.ComplianceMap{
			"zero_a": 0,
			"zero_b": 0,
			"zero_c": 0,
			"zero_d": 0,
			"zero_e": 0,
			"zero_f": 0,
		},
	}}}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	guidance := validGuidanceForAnalysis(analysis)
	guidance.Platforms[0].ZeroGroups[0].Signals[0] = "invented_signal"

	_, err = ParseNarrativeGuidance(mustMarshalGuidance(t, guidance), analysis)
	if err == nil || !strings.Contains(err.Error(), `unexpected signal "invented_signal"`) {
		t.Fatalf("error = %v, want strict unknown zero-signal rejection", err)
	}
}

func TestParseNarrativeGuidanceAcceptsUnknownPresentationFields(t *testing.T) {
	analysis := testGuidanceAnalysis(t)
	raw := strings.TrimSuffix(mustMarshalGuidance(t, validGuidanceForAnalysis(analysis)), "}") + `,"unexpected":true}`

	if _, err := ParseNarrativeGuidance(raw, analysis); err != nil {
		t.Fatalf("ParseNarrativeGuidance rejected a harmless unknown field: %v", err)
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
