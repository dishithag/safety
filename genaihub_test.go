package summarizer

import (
	"context"
	"errors"
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
	if !strings.Contains(got, "# Zero Trust Assessment Report") || !strings.Contains(got, "**What it is:**") {
		t.Fatalf("Summarize() did not render the expected report:\n%s", got)
	}
	if !strings.Contains(invoker.prompt, "Return exactly one JSON object") {
		t.Fatalf("prompt did not contain the structured response contract:\n%s", invoker.prompt)
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
	err := validateGenAIHubStopReason(anthropic.StopReasonMaxTokens, genAIHubMaxTokens)
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
