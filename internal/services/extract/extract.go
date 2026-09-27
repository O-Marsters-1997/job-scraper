// Package extract calls OpenRouter chat completions to turn a user's free
// preference text into stances against the fixed scoring option bank.
package extract

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
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
	return &Client{http: &http.Client{Timeout: 60 * time.Second}, baseURL: chatCompletionsURL}
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

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type jsonSchemaFormat struct {
	Type       string `json:"type"`
	JSONSchema struct {
		Name   string         `json:"name"`
		Strict bool           `json:"strict"`
		Schema map[string]any `json:"schema"`
	} `json:"json_schema"`
}

type chatRequest struct {
	Model          string           `json:"model"`
	Messages       []chatMessage    `json:"messages"`
	ResponseFormat jsonSchemaFormat `json:"response_format"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
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

	reqBody := chatRequest{
		Model:    Model,
		Messages: []chatMessage{{Role: "user", Content: prompt.String()}},
	}
	reqBody.ResponseFormat.Type = "json_schema"
	reqBody.ResponseFormat.JSONSchema.Name = "preference_extraction"
	reqBody.ResponseFormat.JSONSchema.Strict = true
	reqBody.ResponseFormat.JSONSchema.Schema = picksSchema

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal extraction request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build extraction request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("extraction request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return nil, fmt.Errorf("extraction: status %d: %s", resp.StatusCode, string(respBody))
	}

	var decoded chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode extraction response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("extraction: no choices in response")
	}

	var result extractionResult
	if err := json.Unmarshal([]byte(decoded.Choices[0].Message.Content), &result); err != nil {
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
