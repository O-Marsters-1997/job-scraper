package cvedit

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

//go:embed insights.md
var insightsRules string

type InsightsInput struct {
	Bullets        []InsightBullet
	JobDescription string
	JobNote        string
	JobComments    []string
}

type InsightBullet struct {
	SlotID string
	Text   string
	Lean   *float64
}

type InsightsResult struct {
	Lines []InsightLine
	Cost  float64
}

type InsightLine struct {
	SlotID string
	Source string
	Quote  string
	Why    string
}

func (c *Client) Insights(ctx context.Context, apiKey string, in InsightsInput) (InsightsResult, error) {
	req := openrouter.NewRequest(
		Model, "cv_insights", insightsSchema(),
		openrouter.Message{Role: "system", Content: insightsRules},
		openrouter.Message{Role: "user", Content: renderInsights(in)},
	)
	req.Usage = &openrouter.UsageOptions{Include: true}
	req.Reasoning = &openrouter.ReasoningOptions{Effort: "low"}

	reply, err := openrouter.Chat(ctx, c.http, c.baseURL, apiKey, req)
	if err != nil {
		return InsightsResult{Cost: reply.Cost}, fmt.Errorf("insights: %w", err)
	}
	var wire struct {
		Lines []struct {
			SlotID string `json:"slotId"`
			Source string `json:"source"`
			Quote  string `json:"quote"`
			Why    string `json:"why"`
		} `json:"lines"`
	}
	if err := json.Unmarshal([]byte(reply.Content), &wire); err != nil {
		return InsightsResult{Cost: reply.Cost}, fmt.Errorf("decode insights: %w", err)
	}
	lines := make([]InsightLine, len(wire.Lines))
	for i, l := range wire.Lines {
		lines[i] = InsightLine{SlotID: l.SlotID, Source: l.Source, Quote: l.Quote, Why: l.Why}
	}
	return InsightsResult{Lines: lines, Cost: reply.Cost}, nil
}

func renderInsights(in InsightsInput) string {
	var b strings.Builder
	b.WriteString("Bullets:\n")
	for _, bl := range in.Bullets {
		lean := "unknown"
		if bl.Lean != nil {
			lean = fmt.Sprintf("%+.0f%%", *bl.Lean*100)
		}
		fmt.Fprintf(&b, "- [%s] (lean %s) %s\n", bl.SlotID, lean, bl.Text)
	}
	fmt.Fprintf(&b, "\nJob description (source \"jd\"):\n\"\"\"\n%s\n\"\"\"\n", in.JobDescription)
	if in.JobNote != "" {
		fmt.Fprintf(&b, "\nCandidate's note (source \"note\"):\n\"\"\"\n%s\n\"\"\"\n", in.JobNote)
	}
	for i, c := range in.JobComments {
		fmt.Fprintf(&b, "\nCandidate's comment %d (source \"comment\"):\n\"\"\"\n%s\n\"\"\"\n", i+1, c)
	}
	return b.String()
}

func insightsSchema() map[string]any {
	str := map[string]any{"type": "string"}
	line := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"slotId": str,
			"source": map[string]any{"type": "string", "enum": []string{"jd", "note", "comment"}},
			"quote":  str,
			"why":    str,
		},
		"required":             []string{"slotId", "source", "quote", "why"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"lines": map[string]any{"type": "array", "items": line}},
		"required":             []string{"lines"},
		"additionalProperties": false,
	}
}
