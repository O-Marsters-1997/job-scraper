// Package extract calls OpenRouter chat completions to turn a user's free
// preference text into stances against the fixed scoring option bank.
package extract

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

const (
	chatCompletionsURL = "https://openrouter.ai/api/v1/chat/completions"

	// Model is the OpenRouter model requested for every extraction call.
	Model = "openai/gpt-4o-mini"

	maxPicks = 10
)

//go:embed prompt.md
var promptSource string

var promptTemplate = template.Must(template.New("extract-prompt").Parse(promptSource))

// Client calls OpenRouter's chat completions endpoint over HTTP.
type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient() *Client {
	return NewClientAt(chatCompletionsURL, &http.Client{Timeout: 60 * time.Second})
}

// NewClientAt builds a Client for tests to point at an httptest.Server.
func NewClientAt(baseURL string, httpClient *http.Client) *Client {
	return &Client{http: httpClient, baseURL: baseURL}
}

type bankOption struct {
	ID        string
	Label     string
	Dimension string
	Stances   string
}

type promptData struct {
	Bank []bankOption
	Text string
}

type extractedPick struct {
	OptionID string `json:"optionId"`
	Stance   string `json:"stance"`
}

type extractionResult struct {
	Picks []extractedPick `json:"picks"`
}

// Extract sends the rendered prompt and the given bank to OpenRouter, billed
// to apiKey, and parses the structured response into picks. It does not
// check ids or stances against options; that's the caller's job.
func (c *Client) Extract(ctx context.Context, apiKey, text string, options []dto.ScoringOption, dimensions []dto.DimensionSpec) ([]dto.Pick, error) {
	stancesByDim := make(map[dto.Dimension]string, len(dimensions))
	for _, d := range dimensions {
		stancesByDim[d.Key] = strings.Join(d.Stances, "/")
	}
	bank := make([]bankOption, len(options))
	for i, o := range options {
		bank[i] = bankOption{ID: o.ID, Label: o.Label, Dimension: string(o.Dimension), Stances: stancesByDim[o.Dimension]}
	}

	var prompt bytes.Buffer
	if err := promptTemplate.Execute(&prompt, promptData{Bank: bank, Text: text}); err != nil {
		return nil, fmt.Errorf("render extraction prompt: %w", err)
	}

	reply, err := openrouter.Chat(ctx, c.http, c.baseURL, apiKey, openrouter.NewRequest(
		Model, "preference_extraction", picksSchema,
		openrouter.Message{Role: "user", Content: prompt.String()},
	))
	if err != nil {
		return nil, fmt.Errorf("extraction: %w", err)
	}

	var result extractionResult
	if err := json.Unmarshal([]byte(reply.Content), &result); err != nil {
		return nil, fmt.Errorf("decode extraction result: %w", err)
	}

	picks := make([]dto.Pick, 0, len(result.Picks))
	for _, p := range result.Picks {
		picks = append(picks, dto.Pick{OptionID: p.OptionID, Stance: p.Stance, Source: "text"})
	}
	return picks, nil
}

var picksSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"picks": map[string]any{
			"type":     "array",
			"maxItems": maxPicks,
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"optionId": map[string]any{"type": "string"},
					"stance":   map[string]any{"type": "string"},
				},
				"required":             []string{"optionId", "stance"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"picks"},
	"additionalProperties": false,
}
