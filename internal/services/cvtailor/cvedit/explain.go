package cvedit

import (
	"context"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

const explainRules = `You help a candidate see why one CV bullet scored low against a job. A separate
classifier gave the three probabilities; you cannot see how it decided, so you are guessing.
Reply with one sentence naming what the job asks for that the bullet does not show. Do not
rewrite the bullet, do not suggest facts the candidate might add, and do not mention the
probabilities.`

// ExplainInput is one low-bullet question. Probabilities are Jev's answer to
// whether the job would value a candidate with the bullet.
type ExplainInput struct {
	Bullet         string
	JobDescription string
	PYes, PNo      float64
	PNotStated     float64
}

type ExplainResult struct {
	Text string
	Cost float64
}

// Explain asks Haiku, after the fact, why the bullet scored low.
func (c *Client) Explain(ctx context.Context, apiKey string, in ExplainInput) (ExplainResult, error) {
	task := fmt.Sprintf("Bullet:\n%s\n\nJob description:\n%s\n\nProbabilities that the job values the bullet: yes %.2f, no %.2f, not stated %.2f\n",
		in.Bullet, in.JobDescription, in.PYes, in.PNo, in.PNotStated)
	reply, err := openrouter.Chat(ctx, c.http, c.baseURL, apiKey, openrouter.Request{
		Model: SuggestModel,
		Messages: []openrouter.Message{
			{Role: "system", Content: explainRules},
			{Role: "user", Content: task},
		},
		Usage: &openrouter.UsageOptions{Include: true},
	})
	if err != nil {
		return ExplainResult{Cost: reply.Cost}, fmt.Errorf("explain: %w", err)
	}
	return ExplainResult{Text: strings.TrimSpace(reply.Content), Cost: reply.Cost}, nil
}
