package score

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	defaultModelID        = "claude-haiku-4-5-20251001"
	defaultMaxInputTokens = 4096

	// haiku4_5 pricing (per million tokens, USD)
	haiku4_5InputPricePerMToken  = 0.80
	haiku4_5OutputPricePerMToken = 4.00
)

// ClaudeScorer calls the Anthropic Messages API to score a job's suitability.
type ClaudeScorer struct {
	client         anthropic.Client
	modelID        string
	maxInputTokens int
}

type ClaudeScorerConfig struct {
	APIKey         string
	ModelID        string // defaults to claude-haiku-4-5-20251001
	MaxInputTokens int    // defaults to 4096
}

func NewClaudeScorer(cfg ClaudeScorerConfig) *ClaudeScorer {
	modelID := cfg.ModelID
	if modelID == "" {
		modelID = defaultModelID
	}
	maxInputTokens := cfg.MaxInputTokens
	if maxInputTokens <= 0 {
		maxInputTokens = defaultMaxInputTokens
	}
	return &ClaudeScorer{
		client:         anthropic.NewClient(option.WithAPIKey(cfg.APIKey)),
		modelID:        modelID,
		maxInputTokens: maxInputTokens,
	}
}

// claudeResponse is the JSON shape expected from the model.
type claudeResponse struct {
	Score     int      `json:"score"`
	Matched   []string `json:"matched"`
	Missing   []string `json:"missing"`
	Rationale string   `json:"rationale"`
}

func (c *ClaudeScorer) Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig) (SuitabilityResult, error) {
	desc := truncate(job.Description, c.maxInputTokens*4)

	prompt := buildPrompt(job, desc, cfg)

	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.modelID),
		MaxTokens: 512,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return SuitabilityResult{}, fmt.Errorf("anthropic messages.new: %w", err)
	}

	raw := strings.TrimSpace(msg.Content[0].Text)
	var resp claudeResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return SuitabilityResult{}, fmt.Errorf("parse score response %q: %w", raw, err)
	}
	if resp.Score < 0 {
		resp.Score = 0
	}
	if resp.Score > 100 {
		resp.Score = 100
	}

	usage := TokenUsage{
		InputTokens:  int(msg.Usage.InputTokens),
		OutputTokens: int(msg.Usage.OutputTokens),
		CostUSD:      costUSD(msg.Usage.InputTokens, msg.Usage.OutputTokens),
	}

	slog.Info("suitability scored via claude",
		slog.String("url", job.URL),
		slog.Int("score", resp.Score),
		slog.Int("input_tokens", usage.InputTokens),
		slog.Int("output_tokens", usage.OutputTokens),
		slog.Float64("cost_usd", usage.CostUSD),
	)

	return SuitabilityResult{
		Score:     resp.Score,
		Matched:   resp.Matched,
		Missing:   resp.Missing,
		Rationale: resp.Rationale,
		Usage:     usage,
	}, nil
}

func buildPrompt(job dto.Job, desc string, cfg dto.SearchConfig) string {
	var sb strings.Builder
	sb.WriteString("Score the following job posting for suitability on a scale of 0-100.\n\n")
	if cfg.SuitabilityRubric != "" {
		sb.WriteString("Rubric:\n")
		sb.WriteString(cfg.SuitabilityRubric)
		sb.WriteString("\n\n")
	}
	sb.WriteString("Job title: ")
	sb.WriteString(job.Title)
	sb.WriteString("\nJob URL: ")
	sb.WriteString(job.URL)
	sb.WriteString("\nDescription:\n")
	sb.WriteString(desc)
	sb.WriteString("\n\nRespond ONLY with valid JSON in this exact shape (no markdown, no extra text):\n")
	sb.WriteString(`{"score":<int 0-100>,"matched":[<skills/criteria present>],"missing":[<skills/criteria absent>],"rationale":"<one sentence>"}`)
	return sb.String()
}

// truncate caps s to at most maxChars characters (bytes), preserving valid UTF-8.
func truncate(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars]
}

func costUSD(inputTokens, outputTokens int64) float64 {
	return float64(inputTokens)/1_000_000*haiku4_5InputPricePerMToken +
		float64(outputTokens)/1_000_000*haiku4_5OutputPricePerMToken
}
