package summarizer

import (
	"reflect"
	"strings"
	"testing"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestAnalyzeReportSeparatesZerosAndIncludesCutoffTies(t *testing.T) {
	report := &shared.CIDReport{
		CID:                      "cid-ties",
		NumAIDs:                  12,
		AverageOverallScore:      45.678,
		AverageOSScore:           40.123,
		AverageSensorConfigScore: 51.234,
		Platforms: []shared.PlatformSummary{
			{
				Name:                     "Windows 11",
				NumAIDs:                  12,
				AverageOverallScore:      45.678,
				AverageOSScore:           40.123,
				AverageSensorConfigScore: 51.234,
				Compliance: shared.ComplianceMap{
					"zero_b": 0,
					"zero_a": 0,
					"tie_e":  0.011,
					"tie_c":  0.011,
					"tie_a":  0.011,
					"tie_d":  0.011,
					"tie_b":  0.011,
					"next":   0.012,
					"full":   1,
				},
			},
		},
	}

	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	platform := analysis.Platforms[0]

	if got, want := signalNames(platform.ZeroSignals), []string{"zero_a", "zero_b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("zero signals = %v, want %v", got, want)
	}
	if got, want := signalNames(platform.PrioritySignals), []string{"tie_a", "tie_b", "tie_c", "tie_d", "tie_e"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("priority signals = %v, want all exact cutoff ties %v", got, want)
	}
	if platform.FullComplianceCount != 1 {
		t.Fatalf("FullComplianceCount = %d, want 1", platform.FullComplianceCount)
	}
	if analysis.AverageOverallScore != 45.68 {
		t.Fatalf("AverageOverallScore = %.2f, want 45.68", analysis.AverageOverallScore)
	}
}

func TestAnalyzeReportUsesRawComplianceForTieCutoff(t *testing.T) {
	report := &shared.CIDReport{Platforms: []shared.PlatformSummary{{
		Name: "Windows 11",
		Compliance: shared.ComplianceMap{
			"first":  0.01000,
			"second": 0.01001,
			"third":  0.01002,
			"fourth": 0.01003,
		},
	}}}

	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	if got, want := signalNames(analysis.Platforms[0].PrioritySignals), []string{"first", "second", "third"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("priority signals = %v, want %v", got, want)
	}
}

func TestAnalyzeReportHandlesManyZerosFullAndNearFullPlatforms(t *testing.T) {
	zeros := shared.ComplianceMap{}
	for _, signal := range []string{"a", "b", "c", "d", "e", "f"} {
		zeros[signal] = 0
	}
	report := &shared.CIDReport{Platforms: []shared.PlatformSummary{
		{Name: "Windows 10", Compliance: zeros},
		{Name: "iOS", Compliance: shared.ComplianceMap{"a": 1, "b": 1}},
		{Name: "macOS", Compliance: shared.ComplianceMap{"a": 0.999, "b": 0.9995, "c": 0.9999}},
	}}

	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	if got := len(analysis.Platforms[0].ZeroSignals); got != 6 {
		t.Fatalf("zero signal count = %d, want 6", got)
	}
	if !analysis.Platforms[1].AllSignalsFullyCompliant || len(analysis.Platforms[1].PrioritySignals) != 0 {
		t.Fatalf("fully compliant platform was not recognized: %#v", analysis.Platforms[1])
	}
	if !analysis.Platforms[2].PrioritySignalsNearFull || len(analysis.Platforms[2].PrioritySignals) != 3 {
		t.Fatalf("near-full platform was not recognized: %#v", analysis.Platforms[2])
	}
}

func TestAnalyzeReportRejectsInvalidCompliance(t *testing.T) {
	report := &shared.CIDReport{Platforms: []shared.PlatformSummary{{
		Name:       "Windows 11",
		Compliance: shared.ComplianceMap{"invalid": 1.01},
	}}}

	if _, err := AnalyzeReport(report); err == nil {
		t.Fatal("AnalyzeReport accepted compliance outside 0..1")
	}
}

func TestAnalyzeReportRejectsInvalidScoresAndDeviceCounts(t *testing.T) {
	tests := []struct {
		name   string
		report *shared.CIDReport
	}{
		{name: "score above 100", report: &shared.CIDReport{AverageOverallScore: 100.01}},
		{name: "negative report devices", report: &shared.CIDReport{NumAIDs: -1}},
		{name: "negative platform devices", report: &shared.CIDReport{Platforms: []shared.PlatformSummary{{Name: "Linux", NumAIDs: -1}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := AnalyzeReport(tt.report); err == nil {
				t.Fatal("AnalyzeReport accepted invalid input")
			}
		})
	}
}

func TestAnalyzeReportRepresentative0009CFixture(t *testing.T) {
	report, err := LoadCIDReport("../../testdata/sample_audit_reports/cids/00000000000000000000000000000009c.json")
	if err != nil {
		t.Fatalf("LoadCIDReport returned error: %v", err)
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}

	windows := analysis.Platforms[0]
	if got, want := signalNames(windows.ZeroSignals), []string{"dma_guard_enabled", "real_time_response", "vsm_available"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Windows zero signals = %v, want %v", got, want)
	}
	if got, want := signalNames(windows.PrioritySignals), []string{"credential_guard_running", "hvci_enabled", "uefi_memory_protection"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Windows priority signals = %v, want %v", got, want)
	}

	ios := analysis.Platforms[1]
	if len(ios.ZeroSignals) != 0 {
		t.Fatalf("iOS zero signals = %v, want none", signalNames(ios.ZeroSignals))
	}
	if got, want := signalNames(ios.PrioritySignals), []string{"lockdown_mode_enabled_ios", "lock_screen_enabled_ios", "mobile_os_integrity_intact_ios"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("iOS priority signals = %v, want %v", got, want)
	}
}

func TestAnalyzeReportPreservesMissingWindowsSensorScore(t *testing.T) {
	report, err := LoadCIDReport("../../testdata/sample_audit_reports/cids/cc5500000000000000000000000003300.json")
	if err != nil {
		t.Fatalf("LoadCIDReport returned error: %v", err)
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}

	if !analysis.HasOSScore || analysis.HasSensorConfigScore {
		t.Fatalf("report score availability = OS:%t sensor:%t, want OS:true sensor:false", analysis.HasOSScore, analysis.HasSensorConfigScore)
	}
	windows := analysis.Platforms[0]
	if !windows.HasOSScore || windows.HasSensorConfigScore {
		t.Fatalf("Windows score availability = OS:%t sensor:%t, want OS:true sensor:false", windows.HasOSScore, windows.HasSensorConfigScore)
	}

	markdown, err := RenderPlaceholderSummary(analysis)
	if err != nil {
		t.Fatalf("RenderPlaceholderSummary returned error: %v", err)
	}
	if strings.Contains(markdown, "Sensor configuration: 0.00/100") {
		t.Fatalf("missing sensor score rendered as zero:\n%s", markdown)
	}
	if !strings.Contains(markdown, "Sensor configuration: not separately reported") {
		t.Fatalf("missing sensor score was not identified as unavailable:\n%s", markdown)
	}
}

func signalNames(signals []SignalAnalysis) []string {
	names := make([]string, 0, len(signals))
	for _, signal := range signals {
		names = append(names, signal.Signal)
	}
	return names
}
