package cvedit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

type wireMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type wireRequest struct {
	Model          string        `json:"model"`
	Messages       []wireMessage `json:"messages"`
	ResponseFormat struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Name   string `json:"name"`
			Strict bool   `json:"strict"`
			Schema struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"schema"`
		} `json:"json_schema"`
	} `json:"response_format"`
}

func fakeServer(t *testing.T, content string, cost float64, captured *wireRequest) *cvedit.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-or-test" {
			t.Errorf("Authorization = %q, want Bearer sk-or-test", got)
		}
		if err := json.NewDecoder(r.Body).Decode(captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}},
			"usage":   map[string]any{"cost": cost},
		})
	}))
	t.Cleanup(server.Close)
	return cvedit.NewClientAt(server.URL, server.Client())
}

func baseInput() cvedit.Input {
	return cvedit.Input{
		JobDescription: "We need a Go engineer who knows Postgres.",
		Positions: []cvedit.Position{{
			ID: "pos-1", Employer: "Acme", Title: "Engineer",
			SlotTexts:    []string{"Built APIs", "Ran on-call"},
			Achievements: []cvedit.Achievement{{ID: "ach-1", Text: "Cut p99 latency by moving queries to Postgres"}},
		}},
	}
}

func TestClient_Edit_ShapesRequest_DecodesResult(t *testing.T) {
	var captured wireRequest
	content := `{"positions":[{"positionId":"pos-1","bullets":[{"achievement_ids":["ach-1"],"text":"Cut p99 latency"}]}]}`
	client := fakeServer(t, content, 0.0123, &captured)

	res, err := client.Edit(context.Background(), "sk-or-test", baseInput())
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}

	if captured.Model != cvedit.Model {
		t.Errorf("model = %q, want %q", captured.Model, cvedit.Model)
	}
	if len(captured.Messages) != 2 || captured.Messages[0].Role != "system" || captured.Messages[1].Role != "user" {
		t.Fatalf("messages = %+v, want system then user", captured.Messages)
	}
	if !strings.Contains(captured.Messages[0].Content, "Rules:") || !strings.Contains(captured.Messages[0].Content, "Voice:") {
		t.Errorf("system prompt missing voice or rules: %s", captured.Messages[0].Content)
	}
	user := captured.Messages[1].Content
	for _, want := range []string{
		"Go engineer who knows Postgres", "position id=pos-1", "slots=2", "Built APIs", "achievement id=ach-1",
	} {
		if !strings.Contains(user, want) {
			t.Errorf("user prompt missing %q: %s", want, user)
		}
	}
	rf := captured.ResponseFormat
	if rf.Type != "json_schema" || rf.JSONSchema.Name == "" || !rf.JSONSchema.Strict {
		t.Errorf("response format = %+v, want a named strict json_schema", rf)
	}

	if len(res.Edits.Positions) != 1 || res.Edits.Positions[0].Bullets[0].AchievementIDs[0] != "ach-1" {
		t.Errorf("edits = %+v", res.Edits)
	}
	if res.Cost != 0.0123 {
		t.Errorf("cost = %v, want 0.0123", res.Cost)
	}
	if res.Raw != content {
		t.Errorf("raw = %q, want %q", res.Raw, content)
	}
}

func TestClient_Edit_ProfileAndSkillsOnlyWhenSectionsExist(t *testing.T) {
	tests := []struct {
		name                 string
		hasProfile, hasSkill bool
		wantProps            []string
		absentProps          []string
	}{
		{"neither", false, false, []string{"positions"}, []string{"profile", "skills", "jobSkills"}},
		{"profile only", true, false, []string{"positions", "profile"}, []string{"skills", "jobSkills"}},
		{"skills only", false, true, []string{"positions", "skills", "jobSkills"}, []string{"profile"}},
		{"both", true, true, []string{"positions", "profile", "skills", "jobSkills"}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var captured wireRequest
			client := fakeServer(t, `{"positions":[]}`, 0, &captured)
			in := baseInput()
			in.HasProfile, in.BaseProfile = tc.hasProfile, "Old profile"
			in.HasSkills, in.BaseSkills = tc.hasSkill, []string{"Go"}

			if _, err := client.Edit(context.Background(), "sk-or-test", in); err != nil {
				t.Fatalf("Edit: %v", err)
			}

			schema := captured.ResponseFormat.JSONSchema.Schema
			for _, p := range tc.wantProps {
				if _, ok := schema.Properties[p]; !ok {
					t.Errorf("schema missing property %q", p)
				}
			}
			for _, p := range tc.absentProps {
				if _, ok := schema.Properties[p]; ok {
					t.Errorf("schema has property %q, want absent", p)
				}
			}
			if len(schema.Required) != len(tc.wantProps) {
				t.Errorf("required = %v, want %v", schema.Required, tc.wantProps)
			}
			user := captured.Messages[1].Content
			if got := strings.Contains(user, "Old profile"); got != tc.hasProfile {
				t.Errorf("prompt mentions profile = %v, want %v", got, tc.hasProfile)
			}
			if got := strings.Contains(user, "Current skills"); got != tc.hasSkill {
				t.Errorf("prompt mentions skills = %v, want %v", got, tc.hasSkill)
			}
		})
	}
}

func TestClient_Edit_RetryIncludesFindingsAndShorten(t *testing.T) {
	var captured wireRequest
	client := fakeServer(t, `{"positions":[]}`, 0, &captured)
	in := baseInput()
	in.PriorEdits = &cvedit.EditSet{Positions: []cvedit.PositionEdit{{PositionID: "pos-1"}}}
	in.PriorFindings = []checks.Finding{{Check: "grounding", SlotID: "s1", Message: "40% is not in the achievement"}}
	in.ShortenBullets = []string{"A very long bullet about latency"}

	if _, err := client.Edit(context.Background(), "sk-or-test", in); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	user := captured.Messages[1].Content
	for _, want := range []string{"previous answer", "[grounding] slot=s1: 40% is not in the achievement", "Shorten these bullets", "A very long bullet about latency"} {
		if !strings.Contains(user, want) {
			t.Errorf("retry prompt missing %q: %s", want, user)
		}
	}
}

func TestClient_Edit_FirstAttemptOmitsRetrySections(t *testing.T) {
	var captured wireRequest
	client := fakeServer(t, `{"positions":[]}`, 0, &captured)

	if _, err := client.Edit(context.Background(), "sk-or-test", baseInput()); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	user := captured.Messages[1].Content
	for _, unwanted := range []string{"previous answer", "failed these checks", "Shorten"} {
		if strings.Contains(user, unwanted) {
			t.Errorf("first-attempt prompt contains %q", unwanted)
		}
	}
}

func TestClient_Edit_DecodesProfileSkillsAndJobSkills(t *testing.T) {
	var captured wireRequest
	client := fakeServer(t, `{"positions":[],"profile":"New profile","skills":["Go"],"jobSkills":["Go","Postgres"]}`, 0, &captured)

	res, err := client.Edit(context.Background(), "sk-or-test", baseInput())
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if res.Edits.Profile == nil || *res.Edits.Profile != "New profile" {
		t.Errorf("profile = %v", res.Edits.Profile)
	}
	if len(res.Edits.Skills) != 1 || len(res.Edits.JobSkills) != 2 {
		t.Errorf("skills = %v, jobSkills = %v", res.Edits.Skills, res.Edits.JobSkills)
	}
}

func TestClient_Edit_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := cvedit.NewClientAt(server.URL, server.Client())

	_, err := client.Edit(context.Background(), "sk-or-test", baseInput())
	if want := fmt.Sprintf("status %d", http.StatusInternalServerError); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error = %v, want it to mention %q", err, want)
	}
}

func TestClient_Edit_MalformedContent_KeepsRawAndErrors(t *testing.T) {
	var captured wireRequest
	client := fakeServer(t, `not json`, 0.5, &captured)

	res, err := client.Edit(context.Background(), "sk-or-test", baseInput())
	if err == nil {
		t.Fatal("Edit: want error, got nil")
	}
	if res.Raw != "not json" || res.Cost != 0.5 {
		t.Errorf("result = %+v, want raw and cost kept for the caller to record", res)
	}
}
