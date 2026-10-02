package cvedit

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

// SuggestModel is the OpenRouter ID of the model that writes inline
// suggestions.
const SuggestModel = "anthropic/claude-haiku-4.5"

//go:embed suggest.md
var suggestRules string

// SuggestPromptVersion identifies the embedded voice.md and suggest.md pair.
var SuggestPromptVersion = hashPrompts(voice, suggestRules)

// SuggestInput is one inline-edit request for a single line. MaxChars applies
// to the fit action; Prompt to ask.
type SuggestInput struct {
	Action       string
	Prompt       string
	Text         string
	MaxChars     int
	Achievements []string
}

type SuggestResult struct {
	Text string
	Cost float64
}

// Suggest streams a rewrite of in.Text to onDelta and returns it whole.
func (c *Client) Suggest(ctx context.Context, apiKey string, in SuggestInput, onDelta func(string)) (SuggestResult, error) {
	task, err := renderSuggest(in)
	if err != nil {
		return SuggestResult{}, err
	}
	req := openrouter.Request{
		Model: SuggestModel,
		Messages: []openrouter.Message{
			{Role: "system", Content: voice + "\n" + suggestRules},
			{Role: "user", Content: task},
		},
		Usage: &openrouter.UsageOptions{Include: true},
	}
	reply, err := openrouter.ChatStream(ctx, c.http, c.baseURL, apiKey, req, onDelta)
	if err != nil {
		return SuggestResult{Cost: reply.Cost}, fmt.Errorf("suggest: %w", err)
	}
	return SuggestResult{Text: strings.TrimSpace(reply.Content), Cost: reply.Cost}, nil
}

func renderSuggest(in SuggestInput) (string, error) {
	var b strings.Builder
	switch in.Action {
	case "fit":
		fmt.Fprintf(&b, "Shorten the line to at most %d characters, keeping its meaning and the facts it draws on.\n", in.MaxChars)
	case "tighten":
		b.WriteString("Tighten the line: cut filler words and keep every fact. It must not get longer.\n")
	case "verb":
		b.WriteString("Start the line with a stronger, more specific verb. Change nothing else.\n")
	case "ask":
		fmt.Fprintf(&b, "Rewrite the line as the person asks, within the rules: %s\n", in.Prompt)
	default:
		return "", fmt.Errorf("unknown suggest action %q", in.Action)
	}
	fmt.Fprintf(&b, "\nLine:\n%s\n\nAchievements it may draw on:\n", in.Text)
	for _, a := range in.Achievements {
		fmt.Fprintf(&b, "- %s\n", a)
	}
	return b.String(), nil
}
