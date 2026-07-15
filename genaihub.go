package summarizer

import (
	"context"
	"fmt"
	"os"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

const genAIHubMaxTokens int64 = 16384

// GenAIHubGenerator turns a CID report into Markdown by calling GenAI Hub.
type GenAIHubGenerator struct {
	model   string
	invoker textInvoker
}

// NewGenAIHubGenerator creates the live GenAI Hub-backed narrative generator.
func NewGenAIHubGenerator(cfg *Config) (*GenAIHubGenerator, error) {
	if strings.TrimSpace(cfg.GenAIHubModel) == "" {
		return nil, fmt.Errorf("create GenAI Hub generator: GENAI_HUB_MODEL is not set")
	}
	baseURL := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if baseURL == "" {
		return nil, fmt.Errorf("create GenAI Hub generator: ANTHROPIC_BASE_URL is not set")
	}
	authToken := strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN"))
	if authToken == "" {
		return nil, fmt.Errorf("create GenAI Hub generator: ANTHROPIC_AUTH_TOKEN is not set")
	}

	client := anthropic.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAuthToken(authToken),
	)
	return newGenAIHubGenerator(cfg.GenAIHubModel, textInvokerFunc(func(ctx context.Context, prompt string) (string, error) {
		response, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:       anthropic.Model(cfg.GenAIHubModel),
			MaxTokens:   genAIHubMaxTokens,
			Temperature: anthropic.Float(0),
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return "", err
		}
		if err := validateGenAIHubStopReason(response.StopReason, response.Usage.OutputTokens); err != nil {
			return "", err
		}

		var b strings.Builder
		for _, block := range response.Content {
			b.WriteString(block.Text)
		}
		return b.String(), nil
	}))
}

func validateGenAIHubStopReason(reason anthropic.StopReason, outputTokens int64) error {
	switch reason {
	case anthropic.StopReasonMaxTokens:
		return fmt.Errorf("model response was truncated after %d output tokens; increase the output limit or reduce the requested report detail", outputTokens)
	case anthropic.StopReasonRefusal:
		return fmt.Errorf("model refused the guidance request")
	default:
		return nil
	}
}

// Summarize asks GenAI Hub for structured guidance and renders deterministic Markdown.
func (g *GenAIHubGenerator) Summarize(ctx context.Context, report *shared.CIDReport) (string, error) {
	analysis, err := AnalyzeReport(report)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: %w", err)
	}
	prompt, err := BuildGuidancePrompt(analysis)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: %w", err)
	}

	response, err := g.invoker.InvokeWithText(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: invoke model %s: %w", g.model, err)
	}

	guidance, err := ParseNarrativeGuidance(response, analysis)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: model %s returned invalid guidance: %w", g.model, err)
	}

	markdown, err := RenderNarrativeMarkdown(analysis, guidance)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: %w", err)
	}
	return markdown, nil
}

func newGenAIHubGenerator(model string, invoker textInvoker) (*GenAIHubGenerator, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("create GenAI Hub generator: GENAI_HUB_MODEL is not set")
	}
	if invoker == nil {
		return nil, fmt.Errorf("create GenAI Hub generator: text invoker is nil")
	}

	return &GenAIHubGenerator{
		model:   model,
		invoker: invoker,
	}, nil
}

type textInvoker interface {
	InvokeWithText(ctx context.Context, prompt string) (string, error)
}

type textInvokerFunc func(ctx context.Context, prompt string) (string, error)

func (f textInvokerFunc) InvokeWithText(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}
