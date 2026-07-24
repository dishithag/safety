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

// GenAIHubGenerator turns a CID report into Markdown by calling GenAI Hub.
type GenAIHubGenerator struct {
	model               string
	platformsPerRequest int
	invoker             textInvoker
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
	maxOutputTokens := cfg.GenAIHubMaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultGenAIHubMaxOutputTokens
	}
	platformsPerRequest := cfg.GenAIHubPlatformsPerRequest
	if platformsPerRequest <= 0 {
		platformsPerRequest = defaultGenAIHubPlatformsPerRequest
	}

	return newGenAIHubGeneratorWithPlatformLimit(cfg.GenAIHubModel, platformsPerRequest, textInvokerFunc(func(ctx context.Context, prompt string) (string, error) {
		response, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:       anthropic.Model(cfg.GenAIHubModel),
			MaxTokens:   maxOutputTokens,
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
	guidance, err := g.generateGuidance(ctx, analysis)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: %w", err)
	}

	markdown, err := RenderNarrativeMarkdown(analysis, guidance)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: %w", err)
	}
	return markdown, nil
}

func newGenAIHubGenerator(model string, invoker textInvoker) (*GenAIHubGenerator, error) {
	return newGenAIHubGeneratorWithPlatformLimit(model, defaultGenAIHubPlatformsPerRequest, invoker)
}

func newGenAIHubGeneratorWithPlatformLimit(model string, platformsPerRequest int, invoker textInvoker) (*GenAIHubGenerator, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("create GenAI Hub generator: GENAI_HUB_MODEL is not set")
	}
	if platformsPerRequest <= 0 {
		return nil, fmt.Errorf("create GenAI Hub generator: platforms per request must be positive")
	}
	if invoker == nil {
		return nil, fmt.Errorf("create GenAI Hub generator: text invoker is nil")
	}

	return &GenAIHubGenerator{
		model:               model,
		platformsPerRequest: platformsPerRequest,
		invoker:             invoker,
	}, nil
}

func (g *GenAIHubGenerator) generateGuidance(ctx context.Context, analysis *ReportAnalysis) (*NarrativeGuidance, error) {
	if len(analysis.Platforms) == 0 {
		return g.generateGuidanceChunk(ctx, analysis)
	}

	chunkCount := (len(analysis.Platforms) + g.platformsPerRequest - 1) / g.platformsPerRequest
	combined := &NarrativeGuidance{
		Platforms: make([]PlatformGuidance, 0, len(analysis.Platforms)),
	}

	for start := 0; start < len(analysis.Platforms); start += g.platformsPerRequest {
		end := min(start+g.platformsPerRequest, len(analysis.Platforms))
		chunk := reportAnalysisPlatformChunk(analysis, start, end)
		guidance, err := g.generateGuidanceChunk(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf(
				"platform chunk %d of %d (%s): %w",
				start/g.platformsPerRequest+1,
				chunkCount,
				platformNames(chunk.Platforms),
				err,
			)
		}
		combined.Platforms = append(combined.Platforms, guidance.Platforms...)
	}

	if err := validateNarrativeGuidance(combined, analysis); err != nil {
		return nil, fmt.Errorf("validate combined guidance: %w", err)
	}
	return combined, nil
}

func (g *GenAIHubGenerator) generateGuidanceChunk(ctx context.Context, analysis *ReportAnalysis) (*NarrativeGuidance, error) {
	prompt, err := BuildGuidancePrompt(analysis)
	if err != nil {
		return nil, err
	}

	response, err := g.invoker.InvokeWithText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("invoke model %s: %w", g.model, err)
	}

	guidance, err := ParseNarrativeGuidance(response, analysis)
	if err == nil {
		return guidance, nil
	}

	repairPrompt, promptErr := BuildGuidanceRepairPrompt(analysis, response, err)
	if promptErr != nil {
		return nil, fmt.Errorf("model %s returned invalid guidance and repair prompt failed: %v: %w", g.model, err, promptErr)
	}
	repairedResponse, invokeErr := g.invoker.InvokeWithText(ctx, repairPrompt)
	if invokeErr != nil {
		return nil, fmt.Errorf("model %s returned invalid guidance (%v) and repair invocation failed: %w", g.model, err, invokeErr)
	}
	repairedGuidance, repairErr := ParseNarrativeGuidance(repairedResponse, analysis)
	if repairErr != nil {
		return nil, fmt.Errorf("model %s returned invalid guidance after one repair attempt: initial error: %v; repair error: %w", g.model, err, repairErr)
	}
	return repairedGuidance, nil
}

func reportAnalysisPlatformChunk(analysis *ReportAnalysis, start, end int) *ReportAnalysis {
	chunk := *analysis
	chunk.Platforms = append([]PlatformAnalysis(nil), analysis.Platforms[start:end]...)
	return &chunk
}

func platformNames(platforms []PlatformAnalysis) string {
	names := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		names = append(names, platform.Name)
	}
	return strings.Join(names, ", ")
}

type textInvoker interface {
	InvokeWithText(ctx context.Context, prompt string) (string, error)
}

type textInvokerFunc func(ctx context.Context, prompt string) (string, error)

func (f textInvokerFunc) InvokeWithText(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}
