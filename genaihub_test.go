package summarizer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

func TestGenAIHubGeneratorSummarize(t *testing.T) {
	report := &shared.CIDReport{
		CID:                 "cid-123",
		NumAIDs:             17,
		AverageOverallScore: 87.4,
		AverageOSScore:      84.2,
		Platforms: []shared.PlatformSummary{{
			Name:                "Windows 11",
			NumAIDs:             17,
			AverageOverallScore: 87.4,
			AverageOSScore:      84.2,
			Compliance: shared.ComplianceMap{
				"secure_boot_enabled": 0.5,
			},
		}},
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	invoker := &fakeTextInvoker{response: mustMarshalGuidance(t, validGuidanceForAnalysis(analysis))}
	generator, err := newGenAIHubGenerator("claude-test", invoker)
	if err != nil {
		t.Fatalf("newGenAIHubGenerator returned error: %v", err)
	}

	got, err := generator.Summarize(context.Background(), report)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}
	if !strings.Contains(got, "# Zero Trust Assessment Report") || !strings.Contains(got, "- **Why it matters:**") {
		t.Fatalf("Summarize() did not render the expected report:\n%s", got)
	}
	if !strings.Contains(invoker.prompt, "Return exactly one JSON object") {
		t.Fatalf("prompt did not contain the structured response contract:\n%s", invoker.prompt)
	}
}

func TestGenAIHubGeneratorChunksTenPlatformReport(t *testing.T) {
	report, err := LoadCIDReport("../../testdata/sample_audit_reports/cids/ffffffff111122223333444455556666.json")
	if err != nil {
		t.Fatalf("LoadCIDReport returned error: %v", err)
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}

	const platformsPerRequest = 3
	responses := make([]string, 0, 4)
	for start := 0; start < len(analysis.Platforms); start += platformsPerRequest {
		end := min(start+platformsPerRequest, len(analysis.Platforms))
		chunk := reportAnalysisPlatformChunk(analysis, start, end)
		responses = append(responses, mustMarshalGuidance(t, validGuidanceForAnalysis(chunk)))
	}

	invoker := &queuedTextInvoker{responses: responses}
	generator, err := newGenAIHubGeneratorWithPlatformLimit("claude-test", platformsPerRequest, invoker)
	if err != nil {
		t.Fatalf("newGenAIHubGeneratorWithPlatformLimit returned error: %v", err)
	}

	markdown, err := generator.Summarize(context.Background(), report)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}
	if got, want := len(invoker.prompts), 4; got != want {
		t.Fatalf("Claude request count = %d, want %d", got, want)
	}
	if strings.Contains(invoker.prompts[0], `"name": "Windows Server 2022"`) {
		t.Fatalf("first prompt contains a platform from the second chunk:\n%s", invoker.prompts[0])
	}

	previousPosition := -1
	for index, platform := range analysis.Platforms {
		heading := fmt.Sprintf("## %d. %s", index+1, platform.Name)
		position := strings.Index(markdown, heading)
		if position < 0 {
			t.Fatalf("combined Markdown does not contain %q", heading)
		}
		if position <= previousPosition {
			t.Fatalf("platform heading %q is out of order", heading)
		}
		previousPosition = position
	}
	if got := strings.Count(markdown, "# Zero Trust Assessment Report"); got != 1 {
		t.Fatalf("report heading count = %d, want 1", got)
	}
}

func TestGenAIHubGeneratorRejectsInvalidPlatformChunk(t *testing.T) {
	report, err := LoadCIDReport("../../testdata/sample_audit_reports/cids/ffffffff111122223333444455556666.json")
	if err != nil {
		t.Fatalf("LoadCIDReport returned error: %v", err)
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	firstChunk := reportAnalysisPlatformChunk(analysis, 0, 3)
	invoker := &queuedTextInvoker{responses: []string{
		mustMarshalGuidance(t, validGuidanceForAnalysis(firstChunk)),
		`{}`,
		`{}`,
	}}
	generator, err := newGenAIHubGeneratorWithPlatformLimit("claude-test", 3, invoker)
	if err != nil {
		t.Fatalf("newGenAIHubGeneratorWithPlatformLimit returned error: %v", err)
	}

	markdown, err := generator.Summarize(context.Background(), report)
	if err == nil {
		t.Fatal("Summarize accepted an invalid second platform chunk")
	}
	if markdown != "" {
		t.Fatalf("Summarize returned partial Markdown %q", markdown)
	}
	if !strings.Contains(err.Error(), "platform chunk 2 of 4") {
		t.Fatalf("error = %q, want second chunk context", err)
	}
	if got, want := len(invoker.prompts), 3; got != want {
		t.Fatalf("Claude request count after invalid chunk = %d, want %d", got, want)
	}
}

func TestGenAIHubGeneratorRepairsOneInvalidGuidanceResponse(t *testing.T) {
	report := &shared.CIDReport{
		CID: "cid-repair",
		Platforms: []shared.PlatformSummary{{
			Name:       "Windows 11",
			Compliance: shared.ComplianceMap{"secure_boot_enabled": 0.5},
		}},
	}
	analysis, err := AnalyzeReport(report)
	if err != nil {
		t.Fatalf("AnalyzeReport returned error: %v", err)
	}
	invoker := &queuedTextInvoker{responses: []string{
		`{"platforms":[]}`,
		mustMarshalGuidance(t, validGuidanceForAnalysis(analysis)),
	}}
	generator, err := newGenAIHubGenerator("claude-test", invoker)
	if err != nil {
		t.Fatalf("newGenAIHubGenerator returned error: %v", err)
	}

	markdown, err := generator.Summarize(context.Background(), report)
	if err != nil {
		t.Fatalf("Summarize returned error after repair: %v", err)
	}
	if !strings.Contains(markdown, "# Zero Trust Assessment Report") {
		t.Fatalf("repaired response did not produce Markdown:\n%s", markdown)
	}
	if got, want := len(invoker.prompts), 2; got != want {
		t.Fatalf("Claude request count = %d, want initial plus one repair", got)
	}
	if !strings.Contains(invoker.prompts[1], "REPAIR TASK") || !strings.Contains(invoker.prompts[1], "platform count") {
		t.Fatalf("repair prompt lacks validation context:\n%s", invoker.prompts[1])
	}
}

func TestGenAIHubGeneratorSummarizeReturnsInvokerError(t *testing.T) {
	wantErr := errors.New("model unavailable")
	generator, err := newGenAIHubGenerator("claude-test", &fakeTextInvoker{err: wantErr})
	if err != nil {
		t.Fatalf("newGenAIHubGenerator returned error: %v", err)
	}

	_, err = generator.Summarize(context.Background(), &shared.CIDReport{CID: "cid-123"})
	if err == nil {
		t.Fatal("expected Summarize to return an error")
	}
	if !strings.Contains(err.Error(), "invoke model claude-test") {
		t.Fatalf("error = %q, want model context", err.Error())
	}
}

func TestGenAIHubGeneratorSummarizeRejectsEmptyGuidance(t *testing.T) {
	generator, err := newGenAIHubGenerator("claude-test", &fakeTextInvoker{response: "   "})
	if err != nil {
		t.Fatalf("newGenAIHubGenerator returned error: %v", err)
	}

	_, err = generator.Summarize(context.Background(), &shared.CIDReport{CID: "cid-123"})
	if err == nil {
		t.Fatal("expected Summarize to reject empty guidance")
	}
	if !strings.Contains(err.Error(), "returned invalid guidance") {
		t.Fatalf("error = %q, want invalid guidance context", err.Error())
	}
}

func TestValidateGenAIHubStopReasonReportsTruncation(t *testing.T) {
	err := validateGenAIHubStopReason(anthropic.StopReasonMaxTokens, defaultGenAIHubMaxOutputTokens)
	if err == nil || !strings.Contains(err.Error(), "truncated after 16384 output tokens") {
		t.Fatalf("error = %v, want explicit truncation context", err)
	}
}

type fakeTextInvoker struct {
	prompt   string
	response string
	err      error
}

func (f *fakeTextInvoker) InvokeWithText(_ context.Context, prompt string) (string, error) {
	f.prompt = prompt
	return f.response, f.err
}

type queuedTextInvoker struct {
	prompts   []string
	responses []string
}

func (f *queuedTextInvoker) InvokeWithText(_ context.Context, prompt string) (string, error) {
	f.prompts = append(f.prompts, prompt)
	responseIndex := len(f.prompts) - 1
	if responseIndex >= len(f.responses) {
		return "", fmt.Errorf("unexpected invocation %d", responseIndex+1)
	}
	return f.responses[responseIndex], nil
}
