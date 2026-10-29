package extract

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

//go:embed draft_prompt.md
var draftPromptSource string

type draftPromptData struct {
	Dimensions []dto.Dimension
	Phrase     string
}

type draftedQuestion struct {
	Question  string        `json:"question"`
	Dimension dto.Dimension `json:"dimension"`
}

func (c *Client) Draft(ctx context.Context, apiKey, phrase string, dimensions []dto.DimensionSpec) (dto.DraftedQuestion, error) {
	keys := make([]dto.Dimension, len(dimensions))
	enum := make([]string, len(dimensions))
	for i, d := range dimensions {
		keys[i] = d.Key
		enum[i] = string(d.Key)
	}

	var prompt bytes.Buffer
	if err := template.Must(template.New("draft-prompt").Parse(draftPromptSource)).Execute(&prompt, draftPromptData{Dimensions: keys, Phrase: phrase}); err != nil {
		return dto.DraftedQuestion{}, fmt.Errorf("render draft prompt: %w", err)
	}

	reply, err := openrouter.Chat(ctx, c.http, c.baseURL, apiKey, openrouter.NewRequest(
		Model, "custom_option_draft", draftSchema(enum),
		openrouter.Message{Role: "user", Content: prompt.String()},
	))
	if err != nil {
		return dto.DraftedQuestion{}, fmt.Errorf("draft: %w", err)
	}

	var result draftedQuestion
	if err := json.Unmarshal([]byte(reply.Content), &result); err != nil {
		return dto.DraftedQuestion{}, fmt.Errorf("decode draft result: %w", err)
	}
	return dto.DraftedQuestion{Question: result.Question, Dimension: result.Dimension}, nil
}

func draftSchema(dimensions []string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question":  map[string]any{"type": "string"},
			"dimension": map[string]any{"type": "string", "enum": dimensions},
		},
		"required":             []string{"question", "dimension"},
		"additionalProperties": false,
	}
}
