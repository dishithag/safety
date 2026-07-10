package summarizer

import (
	"context"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

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

	client := anthropic.NewClient()
	return newGenAIHubGenerator(cfg.GenAIHubModel, textInvokerFunc(func(ctx context.Context, prompt string) (string, error) {
		response, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(cfg.GenAIHubModel),
			MaxTokens: 2048,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return "", err
		}

		var b strings.Builder
		for _, block := range response.Content {
			b.WriteString(block.Text)
		}
		return b.String(), nil
	}))
}

// Summarize builds a prompt for the report and calls GenAI Hub for Markdown output.
func (g *GenAIHubGenerator) Summarize(ctx context.Context, report *shared.CIDReport) (string, error) {
	prompt, err := BuildNarrativePrompt(report)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: %w", err)
	}

	markdown, err := g.invoker.InvokeWithText(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: invoke model %s: %w", g.model, err)
	}
	if strings.TrimSpace(markdown) == "" {
		return "", fmt.Errorf("GenAI Hub response did not contain Markdown output")
	}

	return strings.TrimSpace(markdown), nil
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
