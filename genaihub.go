package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	shared "go.crwd.dev/ce/zerotrust-analytics/domain"
)

// GenAIHubGenerator turns a CID report into Markdown by calling GenAI Hub over HTTP.
type GenAIHubGenerator struct {
	endpoint string
	model    string
	apiKey   string
	client   *http.Client
}

// NewGenAIHubGenerator creates the live GenAI Hub-backed narrative generator.
func NewGenAIHubGenerator(cfg *Config) (*GenAIHubGenerator, error) {
	if strings.TrimSpace(cfg.GenAIHubEndpoint) == "" {
		return nil, fmt.Errorf("create GenAI Hub generator: GENAI_HUB_ENDPOINT is not set")
	}
	if strings.TrimSpace(cfg.GenAIHubModel) == "" {
		return nil, fmt.Errorf("create GenAI Hub generator: GENAI_HUB_MODEL is not set")
	}

	return &GenAIHubGenerator{
		endpoint: cfg.GenAIHubEndpoint,
		model:    cfg.GenAIHubModel,
		apiKey:   cfg.GenAIHubAPIKey,
		client:   http.DefaultClient,
	}, nil
}

// Summarize builds a prompt for the report and calls GenAI Hub for Markdown output.
func (g *GenAIHubGenerator) Summarize(ctx context.Context, report *shared.CIDReport) (string, error) {
	prompt, err := BuildNarrativePrompt(report)
	if err != nil {
		return "", fmt.Errorf("summarize with GenAI Hub: %w", err)
	}

	return g.generate(ctx, prompt)
}

func (g *GenAIHubGenerator) generate(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(genAIHubRequest{
		Model:  g.model,
		Prompt: prompt,
	})
	if err != nil {
		return "", fmt.Errorf("encode GenAI Hub request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create GenAI Hub request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(g.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+g.apiKey)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call GenAI Hub: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read GenAI Hub response: %w", err)
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("GenAI Hub returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed genAIHubResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", fmt.Errorf("decode GenAI Hub response: %w", err)
	}

	markdown := parsed.ExtractMarkdown()
	if markdown == "" {
		return "", fmt.Errorf("GenAI Hub response did not contain Markdown output")
	}

	return markdown, nil
}

type genAIHubRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type genAIHubResponse struct {
	Markdown string            `json:"markdown"`
	Output   string            `json:"output"`
	Content  string            `json:"content"`
	Choices  []genAIHubChoice  `json:"choices"`
	Messages []genAIHubMessage `json:"messages"`
}

type genAIHubChoice struct {
	Text    string          `json:"text"`
	Message genAIHubMessage `json:"message"`
}

type genAIHubMessage struct {
	Content string `json:"content"`
	Text    string `json:"text"`
}

// ExtractMarkdown extracts the first non-empty text payload from a GenAI Hub response.
func (r genAIHubResponse) ExtractMarkdown() string {
	candidates := []string{r.Markdown, r.Output, r.Content}
	for _, choice := range r.Choices {
		candidates = append(candidates, choice.Text, choice.Message.Content, choice.Message.Text)
	}
	for _, message := range r.Messages {
		candidates = append(candidates, message.Content, message.Text)
	}

	for _, candidate := range candidates {
		if text := strings.TrimSpace(candidate); text != "" {
			return text
		}
	}
	return ""
}
