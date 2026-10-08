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

	"github.com/ollymarsters/job-scraper/internal/docparse"
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
	BaseSkills  []SkillGroup

	PriorEdits     *EditSet
	PriorFindings  []checks.Finding
	ShortenBullets []string
}

// Bullet is one slot's new text; a kept bullet carries its slot's current text.
type Bullet struct {
	Keep           bool     `json:"keep,omitempty"`
	AchievementIDs []string `json:"achievement_ids"`
	Text           string   `json:"text"`
	SlotID         string   `json:"slotId,omitempty"`
}

type PositionEdit struct {
	PositionID string   `json:"positionId"`
	Bullets    []Bullet `json:"bullets"`
}

// SkillGroup is one Skill Line: its label ("" when the base CV has none) and
// its items in order.
type SkillGroup struct {
	Label string   `json:"label"`
	Items []string `json:"items"`
}

// SkillGroups returns the base CV's Skill Lines as groups.
func SkillGroups(s *docparse.SkillsSlot) []SkillGroup {
	groups := make([]SkillGroup, len(s.Lines))
	for i, l := range s.Lines {
		groups[i] = SkillGroup{Label: l.Label, Items: l.Items}
	}
	return groups
}

// CheckLines converts groups to the lines the checks compare.
func CheckLines(groups []SkillGroup) []checks.SkillLine {
	lines := make([]checks.SkillLine, len(groups))
	for i, g := range groups {
		lines[i] = checks.SkillLine{Label: g.Label, Items: g.Items}
	}
	return lines
}

// FlatSkills concatenates the items of every group.
func FlatSkills(groups []SkillGroup) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g.Items...)
	}
	return out
}

// EditSet is the strict JSON shape the model returns. Profile, Skills and
// JobSkills are set only when the input asked for them. LegacySkills marks a
// stored flat skills array, decoded as one unlabelled group.
type EditSet struct {
	Positions    []PositionEdit `json:"positions"`
	Profile      *string        `json:"profile,omitempty"`
	Skills       []SkillGroup   `json:"skills,omitempty"`
	JobSkills    []string       `json:"jobSkills,omitempty"`
	LegacySkills bool           `json:"-"`
}

type editSetJSON EditSet

type editSetWire struct {
	editSetJSON
	Skills json.RawMessage `json:"skills,omitempty"`
}

// DecodeSkills decodes stored skills, either grouped or, for a Draft made
// before Skill Lines, a flat array of items.
func DecodeSkills(raw json.RawMessage) (groups []SkillGroup, legacy bool, err error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, false, nil
	}
	if err := json.Unmarshal(raw, &groups); err == nil {
		return groups, false, nil
	}
	var flat []string
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil, false, fmt.Errorf("decode skills: %w", err)
	}
	return []SkillGroup{{Items: flat}}, true, nil
}

func (e *EditSet) UnmarshalJSON(data []byte) error {
	var w editSetWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	groups, legacy, err := DecodeSkills(w.Skills)
	if err != nil {
		return err
	}
	*e = EditSet(w.editSetJSON)
	e.Skills, e.LegacySkills = groups, legacy
	return nil
}

func (e EditSet) MarshalJSON() ([]byte, error) {
	if !e.LegacySkills {
		return json.Marshal(editSetJSON(e))
	}
	flat, err := json.Marshal(FlatSkills(e.Skills))
	if err != nil {
		return nil, err
	}
	w := editSetWire{editSetJSON: editSetJSON(e)}
	if len(e.Skills) > 0 {
		w.Skills = flat
	}
	return json.Marshal(w)
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
		b.WriteString("\nCurrent skills, one line each as number, label, items:\n")
		for i, g := range in.BaseSkills {
			label := g.Label
			if label == "" {
				label = "(no label)"
			}
			fmt.Fprintf(&b, "%d. %s: %s\n", i+1, label, strings.Join(g.Items, ", "))
		}
		fmt.Fprintf(&b, "Return the skills field with exactly these %d lines, in this order, each with its label copied exactly (an empty string for no label). Reorder each line's items for this job, most relevant first. Keep every item of its own line: drop none, add none, and move none to another line. In jobSkills return at most 8 of the job description's most important requirements: named technologies, languages, frameworks, platforms or tools, and named ways of working such as end-to-end ownership. Leave out generic concepts and nice-to-haves (databases, replication, queuing, distributed systems) and anything already in the current skills, the positions or the profile.\n", len(in.BaseSkills))
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
		props["skills"] = map[string]any{
			"type":     "array",
			"minItems": len(in.BaseSkills),
			"maxItems": len(in.BaseSkills),
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"label": map[string]any{"type": "string"},
					"items": stringArray,
				},
				"required":             []string{"label", "items"},
				"additionalProperties": false,
			},
		}
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
