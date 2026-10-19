package letterwrite

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/openrouter"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

const (
	chatCompletionsURL = "https://openrouter.ai/api/v1/chat/completions"

	Model = cvedit.Model

	maxTokens = 4000
)

//go:embed scaffold.md
var prompt string

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient() *Client {
	return NewClientAt(chatCompletionsURL, &http.Client{Timeout: 3 * time.Minute})
}

func NewClientAt(baseURL string, httpClient *http.Client) *Client {
	return &Client{http: httpClient, baseURL: baseURL}
}

type Achievement struct {
	ID      string
	Text    string
	Context string
}

type Input struct {
	JobDescription   string
	Company          string
	Role             string
	Sections         []string
	SourceParagraphs []string
	Achievements     []Achievement
}

type Bullet struct {
	Text           string   `json:"text"`
	AchievementIDs []string `json:"achievementIds"`
	Gap            bool     `json:"gap"`
}

type Paragraph struct {
	SourceIndex int      `json:"sourceIndex"`
	Purpose     string   `json:"purpose"`
	Bullets     []Bullet `json:"bullets"`
}

type Scaffold struct {
	Paragraphs []Paragraph `json:"paragraphs"`
}

type Result struct {
	Scaffold Scaffold
	Cost     float64
	Raw      string
}

func (c *Client) Scaffold(ctx context.Context, apiKey string, in Input) (Result, error) {
	req := openrouter.NewRequest(
		Model, "cover_letter_scaffold", scaffoldSchema(),
		openrouter.Message{Role: "system", Content: prompt},
		openrouter.Message{Role: "user", Content: renderTask(in)},
	)
	req.Usage = &openrouter.UsageOptions{Include: true}
	req.Reasoning = &openrouter.ReasoningOptions{Effort: "low"}
	req.MaxTokens = maxTokens

	reply, err := openrouter.Chat(ctx, c.http, c.baseURL, apiKey, req)
	if err != nil {
		return Result{}, fmt.Errorf("scaffold: %w", err)
	}

	var s Scaffold
	if err := json.Unmarshal([]byte(reply.Content), &s); err != nil {
		return Result{Raw: reply.Content, Cost: reply.Cost}, fmt.Errorf("decode scaffold: %w", err)
	}
	return Result{Scaffold: s, Cost: reply.Cost, Raw: reply.Content}, nil
}

func renderTask(in Input) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Job: %s at %s\n\nJob description:\n\"\"\"\n%s\n\"\"\"\n\nSection purposes:\n", in.Role, in.Company, in.JobDescription)
	for _, s := range in.Sections {
		fmt.Fprintf(&b, "- %s\n", s)
	}
	b.WriteString("\nAchievements:\n")
	for _, a := range in.Achievements {
		fmt.Fprintf(&b, "- achievement id=%s: %s\n", a.ID, a.Text)
		if a.Context != "" {
			fmt.Fprintf(&b, "  context: %s\n", a.Context)
		}
	}
	b.WriteString("\nSource paragraphs:\n")
	for i, p := range in.SourceParagraphs {
		fmt.Fprintf(&b, "\nparagraph %d:\n%s\n", i, p)
	}
	return b.String()
}

func scaffoldSchema() map[string]any {
	bullet := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text":           map[string]any{"type": "string"},
			"achievementIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"gap":            map[string]any{"type": "boolean"},
		},
		"required":             []string{"text", "achievementIds", "gap"},
		"additionalProperties": false,
	}
	paragraph := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sourceIndex": map[string]any{"type": "integer"},
			"purpose":     map[string]any{"type": "string"},
			"bullets":     map[string]any{"type": "array", "items": bullet},
		},
		"required":             []string{"sourceIndex", "purpose", "bullets"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"paragraphs": map[string]any{"type": "array", "items": paragraph}},
		"required":             []string{"paragraphs"},
		"additionalProperties": false,
	}
}
