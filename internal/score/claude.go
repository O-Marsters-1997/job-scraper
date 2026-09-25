package score

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	DefaultSuitabilityModel = "claude-haiku-4-5-20251001"
	DefaultReasoningModel   = "claude-sonnet-4-6"
	maxDescriptionChars     = 4096 * 4

	haiku4_5InputPricePerMToken      = 0.80
	haiku4_5OutputPricePerMToken     = 4.00
	haiku4_5CacheWritePricePerMToken = haiku4_5InputPricePerMToken * 1.25
	haiku4_5CacheReadPricePerMToken  = haiku4_5InputPricePerMToken * 0.10
)

type ClaudeScorer struct {
	client anthropic.Client
}

func NewClaudeScorer(apiKey string) *ClaudeScorer {
	return &ClaudeScorer{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}
}

type claudeResponse struct {
	Score     int      `json:"score"`
	Matched   []string `json:"matched"`
	Missing   []string `json:"missing"`
	Rationale string   `json:"rationale"`
}

func (c *ClaudeScorer) Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig, modelID string) (SuitabilityResult, error) {
	desc := truncate(stripHTML(job.Description), maxDescriptionChars)

	if modelID == "" {
		modelID = DefaultSuitabilityModel
	}

	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(modelID),
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{
			{
				Text:         buildSystemPrompt(cfg),
				CacheControl: anthropic.NewCacheControlEphemeralParam(),
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buildUserMessage(job, desc))),
		},
	})
	if err != nil {
		return SuitabilityResult{}, fmt.Errorf("anthropic messages.new: %w", err)
	}

	raw := msg.Content[0].Text
	if s, e := strings.Index(raw, "{"), strings.LastIndex(raw, "}"); s >= 0 && e > s {
		raw = raw[s : e+1]
	}
	var resp claudeResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return SuitabilityResult{}, fmt.Errorf("parse score response %q: %w", raw, err)
	}
	resp.Score = max(0, min(100, resp.Score))

	usage := TokenUsage{
		InputTokens:         int(msg.Usage.InputTokens),
		OutputTokens:        int(msg.Usage.OutputTokens),
		CacheCreationTokens: int(msg.Usage.CacheCreationInputTokens),
		CacheReadTokens:     int(msg.Usage.CacheReadInputTokens),
		CostUSD:             costUSD(msg.Usage.InputTokens, msg.Usage.OutputTokens, msg.Usage.CacheCreationInputTokens, msg.Usage.CacheReadInputTokens),
	}

	slog.Info("suitability scored via claude",
		slog.String("url", job.URL),
		slog.Int("score", resp.Score),
		slog.Int("input_tokens", usage.InputTokens),
		slog.Int("output_tokens", usage.OutputTokens),
		slog.Int("cache_creation_tokens", usage.CacheCreationTokens),
		slog.Int("cache_read_tokens", usage.CacheReadTokens),
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

func buildSystemPrompt(cfg dto.SearchConfig) string {
	var sb strings.Builder
	sb.WriteString("Score the following job posting for suitability on a scale of 0-100.\n\n")
	if cfg.SuitabilityRubric != "" {
		sb.WriteString("Rubric:\n")
		sb.WriteString(cfg.SuitabilityRubric)
		sb.WriteString("\n\n")
	}
	sb.WriteString("Respond ONLY with valid JSON in this exact shape (no markdown, no extra text):\n")
	sb.WriteString(`{"score":<int 0-100>,"matched":[<skills/criteria present>],"missing":[<skills/criteria absent>],"rationale":"<one sentence>"}`)
	return sb.String()
}

func buildUserMessage(job dto.Job, desc string) string {
	var sb strings.Builder
	sb.WriteString("Job title: ")
	sb.WriteString(job.Title)
	sb.WriteString("\nJob URL: ")
	sb.WriteString(job.URL)
	sb.WriteString("\nDescription:\n")
	sb.WriteString(desc)
	return sb.String()
}

func truncate(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars]
}

func stripHTML(s string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err != nil {
		return s
	}
	return strings.TrimSpace(doc.Text())
}

func costUSD(inputTokens, outputTokens, cacheWriteTokens, cacheReadTokens int64) float64 {
	return float64(inputTokens)/1_000_000*haiku4_5InputPricePerMToken +
		float64(outputTokens)/1_000_000*haiku4_5OutputPricePerMToken +
		float64(cacheWriteTokens)/1_000_000*haiku4_5CacheWritePricePerMToken +
		float64(cacheReadTokens)/1_000_000*haiku4_5CacheReadPricePerMToken
}
