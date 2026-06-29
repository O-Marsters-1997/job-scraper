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
	defaultModelID        = "claude-haiku-4-5-20251001"
	DefaultReasoningModel = "claude-sonnet-4-6"
	defaultMaxInputTokens = 4096

	haiku4_5InputPricePerMToken      = 0.80
	haiku4_5OutputPricePerMToken     = 4.00
	haiku4_5CacheWritePricePerMToken = haiku4_5InputPricePerMToken * 1.25
	haiku4_5CacheReadPricePerMToken  = haiku4_5InputPricePerMToken * 0.10

	// scoreBatchCap limits jobs per Claude request to bound output tokens and parse blast radius.
	// ponytail: fixed cap; tune if ingest runs are routinely larger.
	scoreBatchCap = 10
)

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

type claudeResponse struct {
	Score     int      `json:"score"`
	Matched   []string `json:"matched"`
	Missing   []string `json:"missing"`
	Rationale string   `json:"rationale"`
}

type batchClaudeResponse struct {
	ID string `json:"id"`
	claudeResponse
}

func (c *ClaudeScorer) Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig, modelID string) (SuitabilityResult, error) {
	desc := truncate(stripHTML(job.Description), c.maxInputTokens*4)

	if modelID == "" {
		modelID = c.modelID
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

// ScoreBatch scores jobs in chunked requests to amortise the cached system-prompt cost.
// Results are in the same order as jobs; any missing from the batch response fall back to Score.
func (c *ClaudeScorer) ScoreBatch(ctx context.Context, jobs []dto.Job, cfg dto.SearchConfig, modelID string) ([]SuitabilityResult, error) {
	if len(jobs) == 0 {
		return nil, nil
	}
	if modelID == "" {
		modelID = c.modelID
	}
	results := make([]SuitabilityResult, 0, len(jobs))
	for i := 0; i < len(jobs); i += scoreBatchCap {
		end := min(i+scoreBatchCap, len(jobs))
		chunk, err := c.scoreBatchChunk(ctx, jobs[i:end], cfg, modelID)
		if err != nil {
			return results, err
		}
		results = append(results, chunk...)
	}
	return results, nil
}

func (c *ClaudeScorer) scoreBatchChunk(ctx context.Context, jobs []dto.Job, cfg dto.SearchConfig, modelID string) ([]SuitabilityResult, error) {
	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(modelID),
		MaxTokens: int64(len(jobs)*180 + 256),
		System: []anthropic.TextBlockParam{
			{
				Text:         buildBatchSystemPrompt(cfg),
				CacheControl: anthropic.NewCacheControlEphemeralParam(),
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buildBatchUserMessage(jobs, c.maxInputTokens*4))),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic batch messages.new: %w", err)
	}

	raw := msg.Content[0].Text
	if s, e := strings.Index(raw, "["), strings.LastIndex(raw, "]"); s >= 0 && e > s {
		raw = raw[s : e+1]
	}

	var batchResps []batchClaudeResponse
	parseErr := json.Unmarshal([]byte(raw), &batchResps)

	usage := TokenUsage{
		InputTokens:         int(msg.Usage.InputTokens),
		OutputTokens:        int(msg.Usage.OutputTokens),
		CacheCreationTokens: int(msg.Usage.CacheCreationInputTokens),
		CacheReadTokens:     int(msg.Usage.CacheReadInputTokens),
		CostUSD:             costUSD(msg.Usage.InputTokens, msg.Usage.OutputTokens, msg.Usage.CacheCreationInputTokens, msg.Usage.CacheReadInputTokens),
	}

	slog.Info("suitability batch scored via claude",
		slog.Int("jobs", len(jobs)),
		slog.Int("input_tokens", usage.InputTokens),
		slog.Int("output_tokens", usage.OutputTokens),
		slog.Int("cache_creation_tokens", usage.CacheCreationTokens),
		slog.Int("cache_read_tokens", usage.CacheReadTokens),
		slog.Float64("cost_usd", usage.CostUSD),
	)

	if parseErr != nil {
		slog.Warn("batch parse failed, falling back to per-job scoring", slog.Any("err", parseErr))
		return c.fallbackScoreEach(ctx, jobs, cfg, modelID)
	}

	byID := make(map[string]batchClaudeResponse, len(batchResps))
	for _, r := range batchResps {
		byID[r.ID] = r
	}

	results := make([]SuitabilityResult, len(jobs))
	var fallbackIdxs []int
	for i, job := range jobs {
		if r, ok := byID[job.ID]; ok {
			results[i] = SuitabilityResult{
				Score:     max(0, min(100, r.Score)),
				Matched:   r.Matched,
				Missing:   r.Missing,
				Rationale: r.Rationale,
				// ponytail: usage is batch-total; per-job breakdown N/A
				Usage: usage,
			}
		} else {
			fallbackIdxs = append(fallbackIdxs, i)
		}
	}

	if len(fallbackIdxs) > 0 {
		slog.Warn("batch response missing jobs, falling back", slog.Int("count", len(fallbackIdxs)))
		for _, idx := range fallbackIdxs {
			r, err := c.Score(ctx, jobs[idx], cfg, modelID)
			if err != nil {
				slog.Warn("fallback score failed", slog.String("url", jobs[idx].URL), slog.Any("err", err))
				continue
			}
			results[idx] = r
		}
	}

	return results, nil
}

func (c *ClaudeScorer) fallbackScoreEach(ctx context.Context, jobs []dto.Job, cfg dto.SearchConfig, modelID string) ([]SuitabilityResult, error) {
	results := make([]SuitabilityResult, 0, len(jobs))
	for _, job := range jobs {
		r, err := c.Score(ctx, job, cfg, modelID)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}
	return results, nil
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

func buildBatchSystemPrompt(cfg dto.SearchConfig) string {
	var sb strings.Builder
	sb.WriteString("Score each job posting for suitability on a scale of 0-100.\n\n")
	if cfg.SuitabilityRubric != "" {
		sb.WriteString("Rubric:\n")
		sb.WriteString(cfg.SuitabilityRubric)
		sb.WriteString("\n\n")
	}
	sb.WriteString("Respond ONLY with a valid JSON array, one object per job in input order (no markdown, no extra text):\n")
	sb.WriteString(`[{"id":"<job id>","score":<int 0-100>,"matched":[<skills present>],"missing":[<skills absent>],"rationale":"<one sentence>"},...]`)
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

func buildBatchUserMessage(jobs []dto.Job, maxCharsEach int) string {
	var sb strings.Builder
	for i, job := range jobs {
		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		fmt.Fprintf(&sb, "Job id: %s\nJob title: %s\nJob URL: %s\nDescription:\n%s",
			job.ID, job.Title, job.URL, truncate(stripHTML(job.Description), maxCharsEach))
	}
	return sb.String()
}

func truncate(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars]
}

// Block elements may word-join without spaces; upgrade to block-aware stripping if scoring quality degrades.
// ponytail: naive .Text(); acceptable for scoring, not for display.
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
