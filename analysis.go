package summarizer

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

const (
	fullCompliancePercent       = 100.0
	nearFullCompliancePercent   = 99.0
	maxIndividualZeroGapDetails = 5
)

// ReportAnalysis contains the deterministic facts used to render a narrative.
type ReportAnalysis struct {
	CID                      string             `json:"cid"`
	ReportedDevices          int                `json:"reported_devices"`
	AverageOverallScore      float64            `json:"average_overall_score"`
	AverageOSScore           float64            `json:"average_os_score"`
	AverageSensorConfigScore float64            `json:"average_sensor_config_score"`
	HasOSScore               bool               `json:"has_os_score"`
	HasSensorConfigScore     bool               `json:"has_sensor_config_score"`
	Platforms                []PlatformAnalysis `json:"platforms"`
}

// PlatformAnalysis contains a platform's scores and preselected findings.
type PlatformAnalysis struct {
	Name                     string           `json:"name"`
	ReportedDevices          int              `json:"reported_devices"`
	AverageOverallScore      float64          `json:"average_overall_score"`
	AverageOSScore           float64          `json:"average_os_score"`
	AverageSensorConfigScore float64          `json:"average_sensor_config_score"`
	HasOSScore               bool             `json:"has_os_score"`
	HasSensorConfigScore     bool             `json:"has_sensor_config_score"`
	SignalCount              int              `json:"signal_count"`
	FullComplianceCount      int              `json:"full_compliance_count"`
	AllSignalsFullyCompliant bool             `json:"all_signals_fully_compliant"`
	ZeroGuidanceMode         string           `json:"zero_guidance_mode,omitempty"`
	ZeroSignals              []SignalAnalysis `json:"zero_signals"`
	PrioritySignals          []SignalAnalysis `json:"priority_signals"`
	PrioritySignalsNearFull  bool             `json:"priority_signals_near_full"`
}

// SignalAnalysis is a report signal prepared for display and LLM grounding.
type SignalAnalysis struct {
	Signal            string  `json:"signal"`
	DisplayName       string  `json:"display_name"`
	CompliancePercent float64 `json:"compliance_percent"`
}

type scoredSignal struct {
	analysis SignalAnalysis
	raw      float64
}

// AnalyzeReport validates and deterministically selects report findings.
func AnalyzeReport(report *shared.CIDReport) (*ReportAnalysis, error) {
	if report == nil {
		return nil, fmt.Errorf("analyze report: report is nil")
	}
	if report.NumAIDs < 0 {
		return nil, fmt.Errorf("analyze report: reported device count %d is negative", report.NumAIDs)
	}
	if err := validateScores(report.AverageOverallScore, report.AverageOSScore, report.AverageSensorConfigScore); err != nil {
		return nil, fmt.Errorf("analyze report: %w", err)
	}

	analysis := &ReportAnalysis{
		CID:                      report.CID,
		ReportedDevices:          report.NumAIDs,
		AverageOverallScore:      roundScore(report.AverageOverallScore),
		AverageOSScore:           roundScore(report.AverageOSScore),
		AverageSensorConfigScore: roundScore(report.AverageSensorConfigScore),
		Platforms:                make([]PlatformAnalysis, 0, len(report.Platforms)),
	}

	for _, platform := range report.Platforms {
		platformAnalysis, err := analyzePlatform(platform)
		if err != nil {
			return nil, fmt.Errorf("analyze report platform %q: %w", platform.Name, err)
		}
		analysis.Platforms = append(analysis.Platforms, platformAnalysis)
		analysis.HasOSScore = analysis.HasOSScore || platformAnalysis.HasOSScore
		analysis.HasSensorConfigScore = analysis.HasSensorConfigScore || platformAnalysis.HasSensorConfigScore
	}

	if len(report.Platforms) == 0 {
		analysis.HasOSScore = report.AverageOSScore != 0
		analysis.HasSensorConfigScore = report.AverageSensorConfigScore != 0
	}

	return analysis, nil
}

func analyzePlatform(platform shared.PlatformSummary) (PlatformAnalysis, error) {
	if platform.NumAIDs < 0 {
		return PlatformAnalysis{}, fmt.Errorf("reported device count %d is negative", platform.NumAIDs)
	}
	if err := validateScores(platform.AverageOverallScore, platform.AverageOSScore, platform.AverageSensorConfigScore); err != nil {
		return PlatformAnalysis{}, err
	}

	hasOSScore, hasSensorConfigScore := platformScoreAvailability(platform)
	analysis := PlatformAnalysis{
		Name:                     platform.Name,
		ReportedDevices:          platform.NumAIDs,
		AverageOverallScore:      roundScore(platform.AverageOverallScore),
		AverageOSScore:           roundScore(platform.AverageOSScore),
		AverageSensorConfigScore: roundScore(platform.AverageSensorConfigScore),
		HasOSScore:               hasOSScore,
		HasSensorConfigScore:     hasSensorConfigScore,
		SignalCount:              len(platform.Compliance),
		ZeroSignals:              []SignalAnalysis{},
		PrioritySignals:          []SignalAnalysis{},
	}

	partialSignals := make([]scoredSignal, 0, len(platform.Compliance))
	for signal, compliance := range platform.Compliance {
		if math.IsNaN(compliance) || math.IsInf(compliance, 0) || compliance < 0 || compliance > 1 {
			return PlatformAnalysis{}, fmt.Errorf("signal %q compliance %.4f is outside 0..1", signal, compliance)
		}

		prepared := SignalAnalysis{
			Signal:            signal,
			DisplayName:       signalDisplayName(signal),
			CompliancePercent: compliancePercent(compliance),
		}
		switch {
		case compliance == 0:
			analysis.ZeroSignals = append(analysis.ZeroSignals, prepared)
		case compliance == fullCompliancePercent/100:
			analysis.FullComplianceCount++
		default:
			partialSignals = append(partialSignals, scoredSignal{analysis: prepared, raw: compliance})
		}
	}

	sortSignalsAscending(analysis.ZeroSignals)
	sort.Slice(partialSignals, func(i, j int) bool {
		if partialSignals[i].raw == partialSignals[j].raw {
			return partialSignals[i].analysis.Signal < partialSignals[j].analysis.Signal
		}
		return partialSignals[i].raw < partialSignals[j].raw
	})
	analysis.AllSignalsFullyCompliant = analysis.SignalCount > 0 && analysis.FullComplianceCount == analysis.SignalCount
	if len(analysis.ZeroSignals) > 0 {
		analysis.ZeroGuidanceMode = "individual"
		if len(analysis.ZeroSignals) > maxIndividualZeroGapDetails {
			analysis.ZeroGuidanceMode = "grouped"
		}
	}

	if len(partialSignals) > 0 {
		cutoffIndex := min(2, len(partialSignals)-1)
		cutoff := partialSignals[cutoffIndex].raw
		for _, signal := range partialSignals {
			if signal.raw > cutoff {
				break
			}
			analysis.PrioritySignals = append(analysis.PrioritySignals, signal.analysis)
		}
		analysis.PrioritySignalsNearFull = analysis.PrioritySignals[0].CompliancePercent >= nearFullCompliancePercent
	}

	return analysis, nil
}

func platformScoreAvailability(platform shared.PlatformSummary) (bool, bool) {
	name := strings.ToLower(strings.TrimSpace(platform.Name))
	switch name {
	case "android", "ios":
		return true, false
	case "linux":
		return false, true
	}

	if strings.HasPrefix(name, "windows") || name == "macos" {
		return true, true
	}
	return platform.AverageOSScore != 0, platform.AverageSensorConfigScore != 0
}

func sortSignalsAscending(signals []SignalAnalysis) {
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].CompliancePercent == signals[j].CompliancePercent {
			return signals[i].Signal < signals[j].Signal
		}
		return signals[i].CompliancePercent < signals[j].CompliancePercent
	})
}

func compliancePercent(value float64) float64 {
	return math.Round(value*10000) / 100
}

func roundScore(value float64) float64 {
	return math.Round(value*100) / 100
}

func validateScores(overall, os, sensor float64) error {
	scores := []struct {
		name  string
		value float64
	}{
		{name: "average_overall_score", value: overall},
		{name: "average_os_score", value: os},
		{name: "average_sensor_config_score", value: sensor},
	}
	for _, score := range scores {
		name, value := score.name, score.value
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > fullCompliancePercent {
			return fmt.Errorf("%s %.4f is outside 0..100", name, value)
		}
	}
	return nil
}

func signalDisplayName(signal string) string {
	words := strings.Split(signal, "_")
	platformSuffix := ""
	if len(words) > 1 {
		suffixes := map[string]string{
			"android": "Android",
			"ios":     "iOS",
			"lin":     "Linux",
			"mac":     "macOS",
		}
		if platform, ok := suffixes[words[len(words)-1]]; ok {
			platformSuffix = platform
			words = words[:len(words)-1]
		}
	}

	acronyms := map[string]string{
		"amd": "AMD", "aslr": "ASLR", "bios": "BIOS", "cpu": "CPU",
		"dma": "DMA", "hsti": "HSTI", "hvci": "HVCI", "iommu": "IOMMU",
		"ios": "iOS", "kmci": "KMCI", "l1": "L1", "mbec": "MBEC",
		"ml": "ML", "mor": "MOR", "os": "OS", "pcie": "PCIe",
		"rtr": "RTR", "seh": "SEH", "sip": "SIP", "smm": "SMM",
		"uefi": "UEFI", "usb": "USB", "vpn": "VPN", "vsm": "VSM",
	}
	for i, word := range words {
		if acronym, ok := acronyms[word]; ok {
			words[i] = acronym
			continue
		}
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		words[i] = string(runes)
	}

	name := strings.Join(words, " ")
	if platformSuffix != "" {
		name += " (" + platformSuffix + ")"
	}
	return name
}
