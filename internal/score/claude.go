package score

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	defaultModelID       = "claude-haiku-4-5-20251001"
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

func (c *ClaudeScorer) Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig) (int, TokenUsage, error) {
	desc := truncate(job.Description, c.maxInputTokens*4)

	prompt := buildPrompt(job, desc, cfg)

	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.modelID),
		MaxTokens: 16,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return 0, TokenUsage{}, fmt.Errorf("anthropic messages.new: %w", err)
	}

	raw := strings.TrimSpace(msg.Content[0].Text)
	score, err := strconv.Atoi(raw)
	if err != nil {
		return 0, TokenUsage{}, fmt.Errorf("parse score %q: %w", raw, err)
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	usage := TokenUsage{
		InputTokens:  int(msg.Usage.InputTokens),
		OutputTokens: int(msg.Usage.OutputTokens),
		CostUSD:      costUSD(msg.Usage.InputTokens, msg.Usage.OutputTokens),
	}

	slog.Info("suitability scored via claude",
		slog.String("url", job.URL),
		slog.Int("score", score),
		slog.Int("input_tokens", usage.InputTokens),
		slog.Int("output_tokens", usage.OutputTokens),
		slog.Float64("cost_usd", usage.CostUSD),
	)

	return score, usage, nil
}

func buildPrompt(job dto.Job, desc string, cfg dto.SearchConfig) string {
	var sb strings.Builder
	sb.WriteString("Score the following job posting for suitability on a scale of 0-100 (integers only).\n\n")
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
	sb.WriteString("\n\nRespond with a single integer between 0 and 100.")
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
