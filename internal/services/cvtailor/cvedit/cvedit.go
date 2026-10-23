// Package cvedit calls OpenRouter chat completions to rewrite a CV's bullet
// slots for one job from a fixed set of the user's own achievements.
package cvedit

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
)

const (
	chatCompletionsURL = "https://openrouter.ai/api/v1/chat/completions"

	Model = "anthropic/claude-sonnet-5.5"
)

//go:embed voice.md
var voice string

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

// Bullet is one slot's new text; a kept bullet carries its slot's current text.
type Bullet struct {
	Keep           bool     `json:"keep,omitempty"`
	AchievementIDs []string `json:"achievement_ids"`
	Text           string   `json:"text"`
	SlotID         string   `json:"slotId,omitempty"`
}

type PositionEdit struct {
	PositionID string        `json:"positionId"`
	Bullets    []Bullet      `json:"bullets"`
	Added      []AddedBullet `json:"added,omitempty"`
}

type AddedBullet struct {
	SlotID         string   `json:"slotId"`
	After          string   `json:"after"`
	Text           string   `json:"text"`
	AchievementIDs []string `json:"achievement_ids,omitempty"`
}

type Line struct {
	SlotID         string
	Text           string
	AchievementIDs []string
	Added          bool
}

func (pe PositionEdit) Lines() []Line {
	out := make([]Line, 0, len(pe.Bullets)+len(pe.Added))
	var follow func(id string)
	follow = func(id string) {
		for _, a := range pe.Added {
			if a.After == id {
				out = append(out, Line{SlotID: a.SlotID, Text: a.Text, AchievementIDs: a.AchievementIDs, Added: true})
				follow(a.SlotID)
			}
		}
	}
	follow("")
	for _, b := range pe.Bullets {
		out = append(out, Line{SlotID: b.SlotID, Text: b.Text, AchievementIDs: b.AchievementIDs})
		follow(b.SlotID)
	}
	return out
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

// EditSet is the strict JSON shape the model returns. Profile and Skills are
// set only when the input asked for them. LegacySkills marks a stored flat
// skills array, decoded as one unlabelled group.
type EditSet struct {
	Positions    []PositionEdit `json:"positions"`
	Roles        []string       `json:"roles,omitempty"`
	Profile      *string        `json:"profile,omitempty"`
	Skills       []SkillGroup   `json:"skills,omitempty"`
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
