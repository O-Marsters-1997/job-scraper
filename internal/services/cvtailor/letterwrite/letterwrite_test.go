package letterwrite_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/letterwrite"
)

type wireRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Content string `json:"content"`
	} `json:"messages"`
	Usage struct {
		Include bool `json:"include"`
	} `json:"usage"`
	Reasoning struct {
		Effort string `json:"effort"`
	} `json:"reasoning"`
	Format struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Strict bool `json:"strict"`
		} `json:"json_schema"`
	} `json:"response_format"`
}

func fakeServer(t *testing.T, content string, cost float64) (*letterwrite.Client, *wireRequest) {
	t.Helper()
	captured := new(wireRequest)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(captured); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": content}}},
			"usage":   map[string]any{"cost": cost},
		})
	}))
	t.Cleanup(server.Close)
	return letterwrite.NewClientAt(server.URL, server.Client()), captured
}

func input() letterwrite.Input {
	return letterwrite.Input{
		JobDescription:   "Go engineer wanted.",
		Company:          "Acme",
		Role:             "Engineer",
		Sections:         []string{"Why this company", "Relevant experience"},
		SourceParagraphs: []string{"I admire Acme.", "At Globex I built APIs."},
		Achievements: []letterwrite.Achievement{
			{ID: "ach-1", Text: "Cut p99 latency"},
			{ID: "ach-2", Text: "Ran on-call"},
		},
	}
}

func TestClient_Scaffold(t *testing.T) {
	t.Parallel()

	t.Run("shapes the request and decodes the result", func(t *testing.T) {
		t.Parallel()
		content := `{"paragraphs":[{"sourceIndex":1,"purpose":"Relevant experience","bullets":[{"text":"I cut latency","achievementIds":["ach-1"],"gap":false}]}]}`
		client, got := fakeServer(t, content, 0.02)

		res, err := client.Scaffold(t.Context(), "sk-or-test", input())

		if err != nil {
			t.Fatalf("Scaffold() err = %v", err)
		}
		want := letterwrite.Result{
			Scaffold: letterwrite.Scaffold{Paragraphs: []letterwrite.Paragraph{{
				SourceIndex: 1, Purpose: "Relevant experience",
				Bullets: []letterwrite.Bullet{{Text: "I cut latency", AchievementIDs: []string{"ach-1"}}},
			}}},
			Cost: 0.02, Raw: content,
		}
		if diff := cmp.Diff(want, res); diff != "" {
			t.Errorf("Scaffold() (-want +got):\n%s", diff)
		}
		if got.Model != letterwrite.Model || got.Reasoning.Effort != "low" || !got.Usage.Include || got.MaxTokens == 0 {
			t.Errorf("request model/reasoning/usage/max_tokens = %q/%q/%v/%d", got.Model, got.Reasoning.Effort, got.Usage.Include, got.MaxTokens)
		}
		if got.Format.Type != "json_schema" || !got.Format.JSONSchema.Strict {
			t.Errorf("response_format = %+v, want strict json_schema", got.Format)
		}
	})

	t.Run("prompt lists achievements flat, section purposes and numbered paragraphs", func(t *testing.T) {
		t.Parallel()
		client, got := fakeServer(t, `{"paragraphs":[]}`, 0)

		if _, err := client.Scaffold(t.Context(), "k", input()); err != nil {
			t.Fatalf("Scaffold() err = %v", err)
		}

		user := got.Messages[1].Content
		for _, want := range []string{
			"- achievement id=ach-1: Cut p99 latency", "- achievement id=ach-2: Ran on-call",
			"- Why this company", "- Relevant experience",
			"paragraph 0:\nI admire Acme.", "paragraph 1:\nAt Globex I built APIs.",
		} {
			if !strings.Contains(user, want) {
				t.Errorf("user message = %q, want it to contain %q", user, want)
			}
		}
	})

	t.Run("malformed output returns an error carrying raw and cost", func(t *testing.T) {
		t.Parallel()
		client, _ := fakeServer(t, `not json`, 0.03)

		res, err := client.Scaffold(t.Context(), "k", input())

		if err == nil {
			t.Fatal("Scaffold() err = nil, want a decode error")
		}
		if res.Raw != "not json" || res.Cost != 0.03 {
			t.Errorf("Scaffold() raw/cost = %q/%v, want not json/0.03", res.Raw, res.Cost)
		}
	})
}
