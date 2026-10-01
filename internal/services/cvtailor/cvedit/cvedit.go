// Package cvedit calls OpenRouter chat completions to rewrite a CV's bullet
// slots for one job from a fixed set of the user's own achievements.
package cvedit

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/openrouter"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
)

const (
	chatCompletionsURL = "https://openrouter.ai/api/v1/chat/completions"

	Model = "anthropic/claude-sonnet-5.5"
)

//go:embed voice.md
var voice string

//go:embed rules.md
var rules string

// PromptVersion identifies the embedded voice.md and rules.md pair.
var PromptVersion = hashPrompts(voice, rules)

func hashPrompts(voice, rules string) string {
	h := sha256.New()
	for _, part := range []string{voice, rules} {
		_, _ = fmt.Fprintf(h, "%d:%s", len(part), part)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient() *Client {
	return NewClientAt(chatCompletionsURL, &http.Client{Timeout: 3 * time.Minute})
}

// NewClientAt builds a Client for tests to point at an httptest.Server.
func NewClientAt(baseURL string, httpClient *http.Client) *Client {
	return &Client{http: httpClient, baseURL: baseURL}
}

type Achievement struct {
	ID   string
	Text string
}

// Position is one role heading in the base CV with its bullet slots.
type Position struct {
	ID           string
	Employer     string
	Title        string
	SlotTexts    []string
	Achievements []Achievement
}

// Input describes one edit request. Profile and skills are asked for only
// when HasProfile / HasSkills are set, i.e. the base CV has those sections.
type Input struct {
	JobDescription string
	Positions      []Position

	HasProfile  bool
	BaseProfile string
	HasSkills   bool
	BaseSkills  []string

	PriorEdits     *EditSet
	PriorFindings  []checks.Finding
	ShortenBullets []string
}

// Bullet is one slot's new text; a kept bullet carries its slot's current text.
type Bullet struct {
	Keep           bool     `json:"keep,omitempty"`
	AchievementIDs []string `json:"achievement_ids"`
	Text           string   `json:"text"`
}

type PositionEdit struct {
	PositionID string   `json:"positionId"`
	Bullets    []Bullet `json:"bullets"`
}

// EditSet is the strict JSON shape the model returns. Profile, Skills and
// JobSkills are set only when the input asked for them.
type EditSet struct {
	Positions []PositionEdit `json:"positions"`
	Profile   *string        `json:"profile,omitempty"`
	Skills    []string       `json:"skills,omitempty"`
	JobSkills []string       `json:"jobSkills,omitempty"`
}

// Result is a decoded edit with the cost OpenRouter billed and the raw
// message content the model produced.
type Result struct {
	Edits EditSet
	Cost  float64
	Raw   string
}

// Edit sends the rendered prompts to OpenRouter, billed to apiKey, and
// decodes the structured response. It does not check the edits against the
// achievements; that's the caller's job.
func (c *Client) Edit(ctx context.Context, apiKey string, in Input) (Result, error) {
	req := openrouter.NewRequest(
		Model, "cv_edit_set", editSchema(in),
		openrouter.Message{Role: "system", Content: voice + "\n" + rules},
		openrouter.Message{Role: "user", Content: renderTask(in)},
	)
	req.Usage = &openrouter.UsageOptions{Include: true}
	req.Reasoning = &openrouter.ReasoningOptions{Effort: "low"}

	reply, err := openrouter.Chat(ctx, c.http, c.baseURL, apiKey, req)
	if err != nil {
		return Result{}, fmt.Errorf("edit: %w", err)
	}

	var edits EditSet
	if err := json.Unmarshal([]byte(reply.Content), &edits); err != nil {
		return Result{Raw: reply.Content, Cost: reply.Cost}, fmt.Errorf("decode edit result: %w", err)
	}
	fillKept(edits, in.Positions)
	return Result{Edits: edits, Cost: reply.Cost, Raw: reply.Content}, nil
}

func fillKept(edits EditSet, positions []Position) {
	slots := make(map[string][]string, len(positions))
	for _, p := range positions {
		slots[p.ID] = p.SlotTexts
	}
	for _, pe := range edits.Positions {
		texts := slots[pe.PositionID]
		for i := range pe.Bullets {
			if pe.Bullets[i].Keep && i < len(texts) {
				pe.Bullets[i].Text = texts[i]
			}
		}
	}
}

func renderTask(in Input) string {
	var b strings.Builder
	b.WriteString("Job description:\n\"\"\"\n")
	b.WriteString(in.JobDescription)
	b.WriteString("\n\"\"\"\n\nPositions. Rewrite the slots using only the listed achievements:\n")
	for _, p := range in.Positions {
		fmt.Fprintf(&b, "\nposition id=%s: %s at %s\nslots=%d\n", p.ID, p.Title, p.Employer, len(p.SlotTexts))
		for i, s := range p.SlotTexts {
			fmt.Fprintf(&b, "current slot %d: %s\n", i+1, s)
		}
		for _, a := range p.Achievements {
			fmt.Fprintf(&b, "- achievement id=%s: %s\n", a.ID, a.Text)
		}
	}
	if in.HasProfile {
		fmt.Fprintf(&b, "\nCurrent profile:\n%s\nRewrite it for this job in the profile field.\n", in.BaseProfile)
	}
	if in.HasSkills {
		fmt.Fprintf(&b, "\nCurrent skills: %s\nReturn the skills list reordered for this job in the skills field, dropping none you cannot support and adding none the person does not already list. Return every skill the job description names in jobSkills.\n", strings.Join(in.BaseSkills, ", "))
	}
	if in.PriorEdits != nil {
		prior, _ := json.Marshal(in.PriorEdits)
		fmt.Fprintf(&b, "\nYour previous answer was:\n%s\n", prior)
	}
	if len(in.PriorFindings) > 0 {
		b.WriteString("\nYour previous answer failed these checks. Fix every one:\n")
		for _, f := range in.PriorFindings {
			fmt.Fprintf(&b, "- [%s] slot=%s: %s\n", f.Check, f.SlotID, f.Message)
		}
	}
	if len(in.ShortenBullets) > 0 {
		b.WriteString("\nThe CV is too long. Shorten these bullets, keeping their citations:\n")
		for _, s := range in.ShortenBullets {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}
	return b.String()
}

func editSchema(in Input) map[string]any {
	stringArray := map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	bullet := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"keep":            map[string]any{"type": "boolean"},
			"achievement_ids": stringArray,
			"text":            map[string]any{"type": "string"},
		},
		"required":             []string{"keep", "achievement_ids", "text"},
		"additionalProperties": false,
	}
	slots := 0
	for _, p := range in.Positions {
		slots = max(slots, len(p.SlotTexts))
	}
	position := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"positionId": map[string]any{"type": "string"},
			"bullets":    map[string]any{"type": "array", "maxItems": slots, "items": bullet},
		},
		"required":             []string{"positionId", "bullets"},
		"additionalProperties": false,
	}
	props := map[string]any{
		"positions": map[string]any{"type": "array", "items": position},
	}
	required := []string{"positions"}
	if in.HasProfile {
		props["profile"] = map[string]any{"type": "string"}
		required = append(required, "profile")
	}
	if in.HasSkills {
		props["skills"] = stringArray
		props["jobSkills"] = stringArray
		required = append(required, "skills", "jobSkills")
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}
