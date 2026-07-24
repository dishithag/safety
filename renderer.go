package summarizer

import (
	"fmt"
	"strconv"
	"strings"
)

type narrativeMode uint8

const (
	narrativeModeGenerated narrativeMode = iota
	narrativeModePlaceholder
	narrativeModeFallback
)

// RenderNarrativeMarkdown validates guidance and renders the final report structure.
func RenderNarrativeMarkdown(analysis *ReportAnalysis, guidance *NarrativeGuidance) (string, error) {
	if analysis == nil {
		return "", fmt.Errorf("render narrative markdown: analysis is nil")
	}
	if guidance == nil {
		return "", fmt.Errorf("render narrative markdown: guidance is nil")
	}
	if err := validateNarrativeGuidance(guidance, analysis); err != nil {
		return "", fmt.Errorf("render narrative markdown: %w", err)
	}
	return renderNarrativeMarkdown(analysis, guidance, narrativeModeGenerated), nil
}

// RenderPlaceholderSummary renders the deterministic report shell without LLM guidance.
func RenderPlaceholderSummary(analysis *ReportAnalysis) (string, error) {
	if analysis == nil {
		return "", fmt.Errorf("render placeholder summary: analysis is nil")
	}
	return renderNarrativeMarkdown(analysis, nil, narrativeModePlaceholder), nil
}

// RenderFallbackSummary renders current report facts when generated guidance is unavailable.
func RenderFallbackSummary(analysis *ReportAnalysis) (string, error) {
	if analysis == nil {
		return "", fmt.Errorf("render fallback summary: analysis is nil")
	}
	return renderNarrativeMarkdown(analysis, nil, narrativeModeFallback), nil
}

func renderNarrativeMarkdown(analysis *ReportAnalysis, guidance *NarrativeGuidance, mode narrativeMode) string {
	var b strings.Builder
	renderReportHeader(&b, analysis)
	if mode == narrativeModeFallback {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "> **Guidance status:** Detailed generated remediation was unavailable for this run. Scores and selected controls below are current source-report facts; generation will be retried.")
	}
	renderHighLevelOverview(&b, analysis)
	renderPlatformAnalysis(&b, analysis, guidance, mode)
	renderRecommendedNextSteps(&b, analysis)
	return strings.TrimSpace(b.String()) + "\n"
}

func renderReportHeader(b *strings.Builder, analysis *ReportAnalysis) {
	fmt.Fprintln(b, "# Zero Trust Assessment Report")
	fmt.Fprintln(b)
	fmt.Fprintf(b, "**CID:** `%s`  \n", inlineCodeValue(analysis.CID))
	fmt.Fprintf(b, "**Reported devices:** **%s**\n", formatInteger(analysis.ReportedDevices))
}

func renderHighLevelOverview(b *strings.Builder, analysis *ReportAnalysis) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, "## High-Level Overview")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "Scores use a **0-100 scale**, with higher values indicating broader compliance with the evaluated security controls.")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "**Score summary**")
	fmt.Fprintln(b)
	fmt.Fprintf(b, "- **Overall posture: %.2f/100.** The report's aggregate endpoint posture score across the represented device population.\n", analysis.AverageOverallScore)
	if analysis.HasOSScore {
		fmt.Fprintf(b, "- **OS security: %.2f/100.** The OS-related component, reflecting native security settings and package-vulnerability posture where evaluated.\n", analysis.AverageOSScore)
	} else {
		fmt.Fprintln(b, "- **OS security: not separately reported.** The OS-related component reflects native security settings and package-vulnerability posture where evaluated.")
	}
	if analysis.HasSensorConfigScore {
		fmt.Fprintf(b, "- **Sensor configuration: %.2f/100.** The Falcon sensor component, reflecting sensor status and configured prevention, detection, and Real Time Response policies where evaluated.\n", analysis.AverageSensorConfigScore)
	} else {
		fmt.Fprintln(b, "- **Sensor configuration: not separately reported.** The Falcon sensor component reflects sensor status and configured prevention, detection, and Real Time Response policies where evaluated.")
	}
	fmt.Fprintln(b)

	switch {
	case analysis.HasOSScore && analysis.HasSensorConfigScore && analysis.AverageOSScore < analysis.AverageSensorConfigScore:
		fmt.Fprint(b, "**Current observation:** OS security is lower than sensor configuration, indicating comparatively lower reported compliance in OS-related controls. ")
	case analysis.HasOSScore && analysis.HasSensorConfigScore && analysis.AverageSensorConfigScore < analysis.AverageOSScore:
		fmt.Fprint(b, "**Current observation:** Sensor configuration is lower than OS security, indicating comparatively lower reported compliance in Falcon sensor-related controls. ")
	case analysis.HasOSScore && analysis.HasSensorConfigScore:
		fmt.Fprint(b, "**Current observation:** OS security and sensor configuration are equal, so the two reported components are balanced at the aggregate level. ")
	case analysis.HasOSScore:
		fmt.Fprint(b, "**Current observation:** OS security is reported separately, but no sensor configuration component is available. ")
	case analysis.HasSensorConfigScore:
		fmt.Fprint(b, "**Current observation:** Sensor configuration is reported separately, but no OS security component is available. ")
	default:
		fmt.Fprint(b, "**Current observation:** The source report does not provide separate OS security or sensor configuration scores. ")
	}
	if len(analysis.Platforms) == 0 {
		fmt.Fprintln(b, "The source report contains no platform-level results.")
		return
	}
	platformWord := pluralize("platform", len(analysis.Platforms))
	fmt.Fprintf(b, "Detailed findings follow for **%d reported %s**.\n", len(analysis.Platforms), platformWord)
}

func renderPlatformAnalysis(b *strings.Builder, analysis *ReportAnalysis, guidance *NarrativeGuidance, mode narrativeMode) {
	if len(analysis.Platforms) == 0 {
		return
	}

	for i := range analysis.Platforms {
		platform := &analysis.Platforms[i]
		var platformGuidance *PlatformGuidance
		if guidance != nil {
			platformGuidance = &guidance.Platforms[i]
		}
		showReportedDevices := len(analysis.Platforms) > 1 || platform.ReportedDevices != analysis.ReportedDevices
		showScoreSummary := len(analysis.Platforms) > 1 || !platformScoresMatchReport(platform, analysis)
		renderPlatform(b, i+1, platform, platformGuidance, mode, showReportedDevices, showScoreSummary)
	}
}

func renderPlatform(b *strings.Builder, index int, platform *PlatformAnalysis, guidance *PlatformGuidance, mode narrativeMode, showReportedDevices, showScoreSummary bool) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, "---")
	fmt.Fprintln(b)
	fmt.Fprintf(b, "## %d. %s\n", index, cleanHeading(platform.Name))
	fmt.Fprintln(b)
	if showReportedDevices {
		fmt.Fprintf(b, "**Reported devices:** **%s**  \n", formatInteger(platform.ReportedDevices))
	}
	if showScoreSummary {
		fmt.Fprintf(b, "**Overall:** **%.2f/100**", platform.AverageOverallScore)
		if platform.HasOSScore {
			fmt.Fprintf(b, " | **OS:** **%.2f/100**", platform.AverageOSScore)
		}
		if platform.HasSensorConfigScore {
			fmt.Fprintf(b, " | **Sensor configuration:** **%.2f/100**", platform.AverageSensorConfigScore)
		}
		fmt.Fprintln(b)
	}
	fmt.Fprintln(b)
	renderPlatformObservation(b, platform)

	if platform.SignalCount == 0 {
		return
	}
	if len(platform.ZeroSignals) > 0 {
		renderZeroComplianceSection(b, platform, guidance, mode)
	}
	if len(platform.PrioritySignals) > 0 {
		renderPrioritySection(b, platform, guidance, mode)
	} else if platform.AllSignalsFullyCompliant {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "No remediation findings were selected for this platform. Continue monitoring for configuration drift.")
	} else if len(platform.ZeroSignals) == platform.SignalCount {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "No partially compliant signals remain after the zero-compliance gaps; the zero-compliance section contains the actionable findings.")
	}
	if mode == narrativeModeGenerated && guidance != nil {
		renderPlatformImplementation(b, guidance)
	}
}

func platformScoresMatchReport(platform *PlatformAnalysis, analysis *ReportAnalysis) bool {
	return platform.AverageOverallScore == analysis.AverageOverallScore &&
		platform.HasOSScore == analysis.HasOSScore &&
		(!platform.HasOSScore || platform.AverageOSScore == analysis.AverageOSScore) &&
		platform.HasSensorConfigScore == analysis.HasSensorConfigScore &&
		(!platform.HasSensorConfigScore || platform.AverageSensorConfigScore == analysis.AverageSensorConfigScore)
}

func renderPlatformObservation(b *strings.Builder, platform *PlatformAnalysis) {
	switch {
	case platform.SignalCount == 0:
		fmt.Fprintln(b, "**Data note:** No per-signal compliance data was reported for this platform, so no control-level findings are generated.")
	case platform.AllSignalsFullyCompliant:
		fmt.Fprintf(b, "**Positive observation:** All **%d tracked signals** report **100%% compliance**.\n", platform.SignalCount)
	case len(platform.ZeroSignals) == 0:
		fmt.Fprintln(b, "**Coverage note:** No signal recorded **0% compliance**.")
	case platform.FullComplianceCount > 0:
		fmt.Fprintf(b, "**Positive observation:** **%d of %d tracked signals** report **100%% compliance**.\n", platform.FullComplianceCount, platform.SignalCount)
	}
}

func renderZeroComplianceSection(b *strings.Builder, platform *PlatformAnalysis, guidance *PlatformGuidance, mode narrativeMode) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, "### Zero-Compliance Gaps")
	fmt.Fprintln(b)
	fmt.Fprintf(
		b,
		"> ZTA recorded **0%% compliance** for **%d %s**. This means no compliant coverage was reported; it does not by itself prove that a control is absent, unsupported, or disabled.\n",
		len(platform.ZeroSignals),
		pluralize("control", len(platform.ZeroSignals)),
	)
	fmt.Fprintln(b)
	fmt.Fprintln(b, "| Control | Signal ID | Reported Compliance |")
	fmt.Fprintln(b, "|---|---|---:|")
	for _, signal := range platform.ZeroSignals {
		fmt.Fprintf(b, "| %s | `%s` | 0.00%% |\n", escapeTable(signal.DisplayName), inlineCodeValue(signal.Signal))
	}

	if mode != narrativeModeGenerated {
		if mode == narrativeModePlaceholder {
			fmt.Fprintln(b)
			fmt.Fprintln(b, "> Technical zero-gap guidance is omitted while `NARRATIVE_PROVIDER=placeholder`.")
		}
		return
	}
	if guidance == nil {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "> Detailed zero-gap guidance is unavailable.")
		return
	}
	for _, group := range guidance.ZeroGroups {
		fmt.Fprintln(b)
		if len(group.Signals) == 1 {
			signal := findSignal(platform.ZeroSignals, group.Signals[0])
			fmt.Fprintf(b, "#### %s - 0.00%%\n", cleanHeading(signal.DisplayName))
		} else {
			fmt.Fprintf(b, "#### Remediation Theme: %s\n", cleanHeading(group.Title))
			fmt.Fprintln(b)
			fmt.Fprintf(b, "**Controls covered:** %s\n", formatSignalReferences(platform.ZeroSignals, group.Signals))
		}
		renderTechnicalGuidance(b, &group.TechnicalGuidance)
	}
}

func renderPrioritySection(b *strings.Builder, platform *PlatformAnalysis, guidance *PlatformGuidance, mode narrativeMode) {
	fmt.Fprintln(b)
	if len(platform.ZeroSignals) > 0 {
		fmt.Fprintln(b, "---")
		fmt.Fprintln(b)
	}
	fmt.Fprintln(b, "### Lowest-Coverage Improvement Opportunities")
	fmt.Fprintln(b)
	if platform.PrioritySignalsNearFull {
		fmt.Fprintln(b, "The remaining partial-compliance findings are near full coverage. Review the outstanding exceptions rather than initiating an unnecessary fleet-wide change.")
	} else {
		fmt.Fprintln(b, "The following controls have the lowest reported non-zero compliance on this platform. This is a coverage-based selection, not a security-severity ranking.")
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, "| Control | Signal ID | Compliance |")
	fmt.Fprintln(b, "|---|---|---:|")
	for _, signal := range platform.PrioritySignals {
		fmt.Fprintf(b, "| %s | `%s` | %.2f%% |\n", escapeTable(signal.DisplayName), inlineCodeValue(signal.Signal), signal.CompliancePercent)
	}

	if mode != narrativeModeGenerated {
		if mode == narrativeModePlaceholder {
			fmt.Fprintln(b)
			fmt.Fprintln(b, "> Technical remediation guidance is omitted while `NARRATIVE_PROVIDER=placeholder`.")
		}
		return
	}
	if guidance == nil {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "> Detailed remediation guidance is unavailable.")
		return
	}
	for i := range guidance.Findings {
		finding := &guidance.Findings[i]
		signal := platform.PrioritySignals[i]
		fmt.Fprintln(b)
		fmt.Fprintf(
			b,
			"#### %s - %.2f%%\n",
			cleanHeading(signal.DisplayName),
			signal.CompliancePercent,
		)
		renderTechnicalGuidance(b, &finding.TechnicalGuidance)
	}
}

func renderTechnicalGuidance(b *strings.Builder, guidance *TechnicalGuidance) {
	fmt.Fprintln(b)
	fmt.Fprintf(b, "- **Why it matters:** %s\n", cleanParagraph(guidance.WhyItMatters))
	if strings.TrimSpace(guidance.SignalInterpretation) != "" {
		fmt.Fprintf(b, "- **Signal interpretation:** %s\n", cleanParagraph(guidance.SignalInterpretation))
	}
	if len(guidance.AdminTerminology) > 0 {
		fmt.Fprintf(b, "- **Also called:** %s\n", formatInlineList(guidance.AdminTerminology))
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, "##### Remediation")
	fmt.Fprintln(b)
	for i, step := range guidance.RemediationSteps {
		fmt.Fprintf(b, "%d. %s\n", i+1, cleanListItem(step))
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, "##### Change Impact")
	fmt.Fprintln(b)
	fmt.Fprintf(
		b,
		"- **Estimated change impact:** **%s** - %s\n",
		guidance.ChangeImpact.Level,
		cleanParagraph(guidance.ChangeImpact.Rationale),
	)
	fmt.Fprintln(b)
	fmt.Fprintln(b, "##### Verification")
	fmt.Fprintln(b)
	for _, step := range guidance.VerificationSteps {
		fmt.Fprintf(b, "- %s\n", cleanListItem(step))
	}
}

func renderPlatformImplementation(b *strings.Builder, guidance *PlatformGuidance) {
	if len(guidance.RemediationSequence) == 0 {
		return
	}

	fmt.Fprintln(b)
	fmt.Fprintln(b, "---")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "### Recommended Remediation Sequence")
	fmt.Fprintln(b)
	for i, step := range guidance.RemediationSequence {
		fmt.Fprintf(b, "%d. %s\n", i+1, cleanListItem(step))
	}

	if len(guidance.SharedBlockers) > 0 {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "#### Shared Blockers")
		for i, blocker := range guidance.SharedBlockers {
			fmt.Fprintln(b)
			fmt.Fprintf(b, "%d. **Blocker:** %s  \n", i+1, cleanParagraph(blocker.Blocker))
			fmt.Fprintf(b, "   **Recommended response:** %s\n", cleanParagraph(blocker.Response))
		}
	}
	if strings.TrimSpace(guidance.FleetGuidance) != "" {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "#### Fleet Rollout Guidance")
		fmt.Fprintln(b)
		fmt.Fprintln(b, cleanParagraph(guidance.FleetGuidance))
	}
}

func renderRecommendedNextSteps(b *strings.Builder, analysis *ReportAnalysis) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, "---")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "## Recommended Next Steps")
	fmt.Fprintln(b)
	for i, step := range deterministicNextSteps(analysis) {
		fmt.Fprintf(b, "%d. %s\n", i+1, step)
	}
}

func deterministicNextSteps(analysis *ReportAnalysis) []string {
	zeroCount := 0
	zeroPlatforms := 0
	priorityCount := 0
	priorityPlatforms := 0
	for _, platform := range analysis.Platforms {
		if len(platform.ZeroSignals) > 0 {
			zeroCount += len(platform.ZeroSignals)
			zeroPlatforms++
		}
		if len(platform.PrioritySignals) > 0 {
			priorityCount += len(platform.PrioritySignals)
			priorityPlatforms++
		}
	}

	var steps []string
	if zeroCount > 0 {
		steps = append(steps, fmt.Sprintf(
			"Confirm whether the %d reported zero-compliance %s across %d %s reflect configuration, platform support, or telemetry availability.",
			zeroCount,
			pluralize("control", zeroCount),
			zeroPlatforms,
			pluralize("platform", zeroPlatforms),
		))
	}
	if priorityCount > 0 {
		steps = append(steps, fmt.Sprintf(
			"Review the %d lowest-coverage partially compliant %s across %d %s and identify shared prerequisites.",
			priorityCount,
			pluralize("control", priorityCount),
			priorityPlatforms,
			pluralize("platform", priorityPlatforms),
		))
	}
	if zeroCount > 0 || priorityCount > 0 {
		steps = append(steps,
			"Sequence shared prerequisites first and pilot potentially disruptive changes on representative devices before broader deployment.",
			"Verify each approved change using the stated success criteria before expanding rollout.",
			"Review the next ZTA assessment for compliance changes without assuming a specific score increase.",
		)
	}
	if len(steps) < 3 {
		steps = append(steps,
			"Continue monitoring fully compliant controls for configuration drift.",
			"Confirm that platform inventory and assessment telemetry remain complete.",
			"Review the next ZTA assessment for newly introduced coverage gaps.",
		)
	}
	return steps[:min(len(steps), 5)]
}

func findSignal(signals []SignalAnalysis, name string) SignalAnalysis {
	for _, signal := range signals {
		if signal.Signal == name {
			return signal
		}
	}
	return SignalAnalysis{Signal: name, DisplayName: signalDisplayName(name)}
}

func formatSignalReferences(available []SignalAnalysis, names []string) string {
	references := make([]string, 0, len(names))
	for _, name := range names {
		signal := findSignal(available, name)
		references = append(references, "**"+cleanParagraph(signal.DisplayName)+"**")
	}
	return strings.Join(references, ", ")
}

func formatInlineList(values []string) string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		cleaned = append(cleaned, cleanParagraph(value))
	}
	return strings.Join(cleaned, "; ")
}

func formatInteger(value int) string {
	negative := value < 0
	digits := strconv.Itoa(value)
	if negative {
		digits = strings.TrimPrefix(digits, "-")
	}
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	if negative {
		return "-" + digits
	}
	return digits
}

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}

func cleanParagraph(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func cleanListItem(value string) string {
	return cleanParagraph(value)
}

func cleanHeading(value string) string {
	return strings.TrimLeft(cleanParagraph(value), "# ")
}

func inlineCodeValue(value string) string {
	return strings.ReplaceAll(cleanParagraph(value), "`", "'")
}

func escapeTable(value string) string {
	value = cleanParagraph(value)
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
