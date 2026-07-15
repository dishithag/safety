package summarizer

import (
	"fmt"
	"strconv"
	"strings"
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
	return renderNarrativeMarkdown(analysis, guidance, false), nil
}

// RenderPlaceholderSummary renders the deterministic report shell without LLM guidance.
func RenderPlaceholderSummary(analysis *ReportAnalysis) (string, error) {
	if analysis == nil {
		return "", fmt.Errorf("render placeholder summary: analysis is nil")
	}
	return renderNarrativeMarkdown(analysis, nil, true), nil
}

func renderNarrativeMarkdown(analysis *ReportAnalysis, guidance *NarrativeGuidance, placeholder bool) string {
	var b strings.Builder
	renderReportHeader(&b, analysis)
	renderHighLevelOverview(&b, analysis)
	renderPlatformAnalysis(&b, analysis, guidance, placeholder)
	renderRecommendedNextSteps(&b, analysis, guidance, placeholder)
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
	fmt.Fprintf(
		b,
		"For the **%s reported %s** represented in the source data, the organization's overall Zero Trust posture score is **%.2f/100**. Scores use a **0-100 scale**, with higher values indicating broader compliance with the evaluated security controls.\n",
		formatInteger(analysis.ReportedDevices),
		pluralize("device", analysis.ReportedDevices),
		analysis.AverageOverallScore,
	)
	fmt.Fprintln(b)
	fmt.Fprintln(b, "**Score summary**")
	fmt.Fprintln(b)
	fmt.Fprintf(b, "- **Overall posture (`average_overall_score`): %.2f/100.** The report's aggregate endpoint posture score across the represented device population.\n", analysis.AverageOverallScore)
	if analysis.HasOSScore {
		fmt.Fprintf(b, "- **OS security (`average_os_score`): %.2f/100.** The OS-related component, reflecting native security settings and package-vulnerability posture where evaluated.\n", analysis.AverageOSScore)
	} else {
		fmt.Fprintln(b, "- **OS security (`average_os_score`): not separately reported.** The OS-related component reflects native security settings and package-vulnerability posture where evaluated.")
	}
	if analysis.HasSensorConfigScore {
		fmt.Fprintf(b, "- **Sensor configuration (`average_sensor_config_score`): %.2f/100.** The Falcon sensor component, reflecting sensor status and configured prevention, detection, and Real Time Response policies where evaluated.\n", analysis.AverageSensorConfigScore)
	} else {
		fmt.Fprintln(b, "- **Sensor configuration (`average_sensor_config_score`): not separately reported.** The Falcon sensor component reflects sensor status and configured prevention, detection, and Real Time Response policies where evaluated.")
	}
	fmt.Fprintln(b)

	switch {
	case analysis.HasOSScore && analysis.HasSensorConfigScore && analysis.AverageOSScore < analysis.AverageSensorConfigScore:
		fmt.Fprintf(
			b,
			"**Current observation:** The **OS security score (%.2f/100)** is lower than the **sensor configuration score (%.2f/100)**, indicating comparatively lower reported compliance in OS-related controls. ",
			analysis.AverageOSScore,
			analysis.AverageSensorConfigScore,
		)
	case analysis.HasOSScore && analysis.HasSensorConfigScore && analysis.AverageSensorConfigScore < analysis.AverageOSScore:
		fmt.Fprintf(
			b,
			"**Current observation:** The **sensor configuration score (%.2f/100)** is lower than the **OS security score (%.2f/100)**, indicating comparatively lower reported compliance in Falcon sensor-related controls. ",
			analysis.AverageSensorConfigScore,
			analysis.AverageOSScore,
		)
	case analysis.HasOSScore && analysis.HasSensorConfigScore:
		fmt.Fprintf(
			b,
			"**Current observation:** The **OS security score** and **sensor configuration score** are both **%.2f/100**, so the two reported components are balanced at the aggregate level. ",
			analysis.AverageOSScore,
		)
	case analysis.HasOSScore:
		fmt.Fprintf(b, "**Current observation:** The source report provides an **OS security score of %.2f/100** and no separate sensor configuration score. ", analysis.AverageOSScore)
	case analysis.HasSensorConfigScore:
		fmt.Fprintf(b, "**Current observation:** The source report provides a **sensor configuration score of %.2f/100** and no separate OS security score. ", analysis.AverageSensorConfigScore)
	default:
		fmt.Fprint(b, "**Current observation:** The source report does not provide separate OS security or sensor configuration scores. ")
	}
	fmt.Fprintln(b, "The platform analysis below identifies the contributing controls and provides risk-aware remediation guidance.")
}

func renderPlatformAnalysis(b *strings.Builder, analysis *ReportAnalysis, guidance *NarrativeGuidance, placeholder bool) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, "## Platform Analysis")
	fmt.Fprintln(b)
	if len(analysis.Platforms) == 0 {
		fmt.Fprintln(b, "The source report contains no platform-level results.")
		return
	}

	platformWord := "platforms"
	if len(analysis.Platforms) == 1 {
		platformWord = "platform"
	}
	fmt.Fprintf(
		b,
		"The source report contains results for **%d %s**: %s. Each section identifies reported strengths, zero-compliance gaps, and the lowest partially compliant controls that warrant review.\n",
		len(analysis.Platforms),
		platformWord,
		formatPlatformNames(analysis.Platforms),
	)

	for i := range analysis.Platforms {
		platform := &analysis.Platforms[i]
		var platformGuidance *PlatformGuidance
		if guidance != nil {
			platformGuidance = &guidance.Platforms[i]
		}
		renderPlatform(b, i+1, platform, platformGuidance, placeholder)
	}
}

func renderPlatform(b *strings.Builder, index int, platform *PlatformAnalysis, guidance *PlatformGuidance, placeholder bool) {
	fmt.Fprintln(b)
	fmt.Fprintf(b, "### %d. %s\n", index, cleanHeading(platform.Name))
	fmt.Fprintln(b)
	fmt.Fprintf(b, "**Reported devices:** **%s**  \n", formatInteger(platform.ReportedDevices))
	fmt.Fprintf(b, "**Overall:** **%.2f/100**", platform.AverageOverallScore)
	if platform.HasOSScore {
		fmt.Fprintf(b, " | **OS:** **%.2f/100**", platform.AverageOSScore)
	}
	if platform.HasSensorConfigScore {
		fmt.Fprintf(b, " | **Sensor configuration:** **%.2f/100**", platform.AverageSensorConfigScore)
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b)
	renderPlatformObservation(b, platform)

	if platform.SignalCount == 0 {
		return
	}
	if len(platform.ZeroSignals) > 0 {
		renderZeroComplianceSection(b, platform, guidance, placeholder)
	}
	if len(platform.PrioritySignals) > 0 {
		renderPrioritySection(b, platform, guidance, placeholder)
	} else if platform.AllSignalsFullyCompliant {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "No remediation findings were selected for this platform. Continue monitoring for configuration drift.")
	} else if len(platform.ZeroSignals) == platform.SignalCount {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "No partially compliant signals remain after the zero-compliance gaps; the zero-compliance section contains the actionable findings.")
	}
}

func renderPlatformObservation(b *strings.Builder, platform *PlatformAnalysis) {
	switch {
	case platform.SignalCount == 0:
		fmt.Fprintln(b, "**Data note:** No per-signal compliance data was reported for this platform, so no control-level findings are generated.")
	case platform.AllSignalsFullyCompliant:
		fmt.Fprintf(b, "**Positive observation:** All **%d tracked signals** report **100%% compliance**.\n", platform.SignalCount)
	case len(platform.ZeroSignals) == 0 && platform.HighestSignal != nil:
		fmt.Fprintf(
			b,
			"**Positive observation:** No signal recorded **0%% compliance**. The highest-performing signal is **%s** (`%s`) at **%.2f%%**.\n",
			platform.HighestSignal.DisplayName,
			inlineCodeValue(platform.HighestSignal.Signal),
			platform.HighestSignal.CompliancePercent,
		)
	case platform.FullComplianceCount > 0:
		fmt.Fprintf(b, "**Positive observation:** **%d of %d tracked signals** report **100%% compliance**.\n", platform.FullComplianceCount, platform.SignalCount)
	}
}

func renderZeroComplianceSection(b *strings.Builder, platform *PlatformAnalysis, guidance *PlatformGuidance, placeholder bool) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, "#### Zero-Compliance Gaps")
	fmt.Fprintln(b)
	fmt.Fprintf(
		b,
		"ZTA recorded **0%% compliance** for **%d %s**. This means no compliant coverage was reported for these signals; it does not by itself prove that a control is absent, uninstalled, unsupported, or disabled.\n",
		len(platform.ZeroSignals),
		pluralize("control", len(platform.ZeroSignals)),
	)
	fmt.Fprintln(b)
	fmt.Fprintln(b, "| Control | Reported Compliance |")
	fmt.Fprintln(b, "|---|---:|")
	for _, signal := range platform.ZeroSignals {
		fmt.Fprintf(b, "| **%s** (`%s`) | **0.00%%** |\n", escapeTable(signal.DisplayName), inlineCodeValue(signal.Signal))
	}

	if placeholder {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "> Technical zero-gap guidance is omitted while `NARRATIVE_PROVIDER=placeholder`.")
		return
	}
	for _, group := range guidance.ZeroGroups {
		fmt.Fprintln(b)
		if len(group.Signals) == 1 {
			signal := findSignal(platform.ZeroSignals, group.Signals[0])
			fmt.Fprintf(b, "##### %s (`%s`) - 0.00%%\n", cleanHeading(signal.DisplayName), inlineCodeValue(signal.Signal))
		} else {
			fmt.Fprintf(b, "##### Remediation Theme: %s\n", cleanHeading(group.Title))
			fmt.Fprintln(b)
			fmt.Fprintf(b, "**Controls covered:** %s\n", formatSignalReferences(platform.ZeroSignals, group.Signals))
		}
		renderTechnicalGuidance(b, &group.TechnicalGuidance)
	}
}

func renderPrioritySection(b *strings.Builder, platform *PlatformAnalysis, guidance *PlatformGuidance, placeholder bool) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, "#### Priority Improvement Opportunities")
	fmt.Fprintln(b)
	if platform.PrioritySignalsNearFull {
		fmt.Fprintln(b, "The remaining partial-compliance findings are near full coverage. Review the outstanding exceptions rather than initiating an unnecessary fleet-wide change.")
	} else {
		fmt.Fprintln(b, "After excluding zero and fully compliant signals, the following are the lowest partially compliant controls. Cutoff ties are included.")
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, "| Control | Compliance |")
	fmt.Fprintln(b, "|---|---:|")
	for _, signal := range platform.PrioritySignals {
		fmt.Fprintf(b, "| **%s** (`%s`) | **%.2f%%** |\n", escapeTable(signal.DisplayName), inlineCodeValue(signal.Signal), signal.CompliancePercent)
	}

	if placeholder {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "> Technical remediation guidance is omitted while `NARRATIVE_PROVIDER=placeholder`.")
		return
	}
	for i := range guidance.Findings {
		finding := &guidance.Findings[i]
		signal := platform.PrioritySignals[i]
		fmt.Fprintln(b)
		fmt.Fprintf(
			b,
			"##### %s (`%s`) - %.2f%%\n",
			cleanHeading(signal.DisplayName),
			inlineCodeValue(signal.Signal),
			signal.CompliancePercent,
		)
		renderTechnicalGuidance(b, &finding.TechnicalGuidance)
	}
}

func renderTechnicalGuidance(b *strings.Builder, guidance *TechnicalGuidance) {
	fmt.Fprintln(b)
	fmt.Fprintf(b, "**What it is:** %s\n", cleanParagraph(guidance.Meaning))
	fmt.Fprintln(b)
	fmt.Fprintf(b, "**Security impact:** %s\n", cleanParagraph(guidance.SecurityImpact))
	fmt.Fprintln(b)
	fmt.Fprintf(b, "**What ZTA evaluates:** %s\n", cleanParagraph(guidance.ZTAEvaluation))
	if strings.TrimSpace(guidance.MeasurementCaveat) != "" {
		fmt.Fprintln(b)
		fmt.Fprintf(b, "**Measurement caveat:** %s\n", cleanParagraph(guidance.MeasurementCaveat))
	}
	fmt.Fprintln(b)
	fmt.Fprintln(b, "**How to improve**")
	fmt.Fprintln(b)
	for i, step := range guidance.RemediationSteps {
		fmt.Fprintf(b, "%d. %s\n", i+1, cleanListItem(step))
	}
	fmt.Fprintln(b)
	fmt.Fprintf(
		b,
		"**Remediation disruption:** **%s** - %s\n",
		guidance.RemediationDisruption.Level,
		cleanParagraph(guidance.RemediationDisruption.Rationale),
	)
	fmt.Fprintln(b)
	fmt.Fprintf(b, "**Operational considerations:** %s\n", cleanParagraph(guidance.OperationalConsiderations))

	if len(guidance.AdminTerminology) > 0 {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "**Administrator terminology**")
		fmt.Fprintln(b)
		for _, term := range guidance.AdminTerminology {
			fmt.Fprintf(b, "- %s\n", cleanListItem(term))
		}
	}
	if len(guidance.Blockers) > 0 {
		fmt.Fprintln(b)
		fmt.Fprintln(b, "**Common blockers**")
		fmt.Fprintln(b)
		fmt.Fprintln(b, "| Blocker | Recommended Response |")
		fmt.Fprintln(b, "|---|---|")
		for _, blocker := range guidance.Blockers {
			fmt.Fprintf(b, "| %s | %s |\n", escapeTable(blocker.Blocker), escapeTable(blocker.Response))
		}
	}

	fmt.Fprintln(b)
	fmt.Fprintln(b, "**Verification**")
	fmt.Fprintln(b)
	for _, step := range guidance.VerificationSteps {
		fmt.Fprintf(b, "- %s\n", cleanListItem(step))
	}
	if strings.TrimSpace(guidance.FleetGuidance) != "" {
		fmt.Fprintln(b)
		fmt.Fprintf(b, "**Fleet guidance:** %s\n", cleanParagraph(guidance.FleetGuidance))
	}
}

func renderRecommendedNextSteps(b *strings.Builder, analysis *ReportAnalysis, guidance *NarrativeGuidance, placeholder bool) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, "## Recommended Next Steps")
	fmt.Fprintln(b)
	if placeholder {
		steps := placeholderNextSteps(analysis)
		for i, step := range steps {
			fmt.Fprintf(b, "%d. %s\n", i+1, step)
		}
		fmt.Fprintln(b)
		fmt.Fprintln(b, "> **Operational tip:** Run with the approved GenAI provider to add platform-specific technical remediation and verification guidance.")
		return
	}

	for i, step := range guidance.RecommendedNextSteps {
		fmt.Fprintf(b, "%d. %s\n", i+1, cleanListItem(step))
	}
	fmt.Fprintln(b)
	fmt.Fprintf(b, "> **Operational tip:** %s\n", cleanParagraph(guidance.OperationalTip))
}

func placeholderNextSteps(analysis *ReportAnalysis) []string {
	hasZeros := false
	hasPriority := false
	for _, platform := range analysis.Platforms {
		hasZeros = hasZeros || len(platform.ZeroSignals) > 0
		hasPriority = hasPriority || len(platform.PrioritySignals) > 0
	}

	var steps []string
	if hasZeros {
		steps = append(steps, "Review the reported zero-compliance signals and confirm whether each gap reflects configuration, support, or data availability.")
	}
	if hasPriority {
		steps = append(steps, "Review the selected partial-compliance controls for each platform and identify shared prerequisites.")
	}
	steps = append(steps,
		"Pilot disruptive configuration changes on representative devices before broader deployment.",
		"Verify each approved change and review the next ZTA report for updated compliance.",
	)
	if len(steps) < 3 {
		steps = append(steps, "Continue monitoring fully compliant controls for configuration drift.")
	}
	return steps
}

func formatPlatformNames(platforms []PlatformAnalysis) string {
	names := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		names = append(names, "**"+cleanParagraph(platform.Name)+"**")
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
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
		references = append(references, fmt.Sprintf("**%s** (`%s`)", cleanParagraph(signal.DisplayName), inlineCodeValue(signal.Signal)))
	}
	return strings.Join(references, ", ")
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
