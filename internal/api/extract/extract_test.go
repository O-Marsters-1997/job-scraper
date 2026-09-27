package extract

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestClient_Extract_SendsPromptAndBank_DecodesPicks(t *testing.T) {
	var captured chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-or-test" {
			t.Errorf("Authorization header = %q, want Bearer sk-or-test", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		content, err := json.Marshal(extractionResult{
			Picks: []extractedPick{{OptionID: "tech:go", Stance: "nice"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		resp := chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: string(content)}}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{http: server.Client(), baseURL: server.URL}

	options := []dto.ScoringOption{{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go"}}
	dimensions := []dto.DimensionSpec{{Key: dto.DimensionTech, Kind: "pair", Stances: []string{"nice", "avoid"}}}

	picks, err := client.Extract(context.Background(), "sk-or-test", "I love Go", options, dimensions)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if captured.Model != Model {
		t.Errorf("model = %q, want %q", captured.Model, Model)
	}
	if len(captured.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(captured.Messages))
	}
	prompt := captured.Messages[0].Content
	if !strings.Contains(prompt, "tech:go") || !strings.Contains(prompt, "Go") {
		t.Errorf("prompt missing bank entry: %s", prompt)
	}
	if !strings.Contains(prompt, "nice/avoid") {
		t.Errorf("prompt missing stances for the bank entry: %s", prompt)
	}
	if !strings.Contains(prompt, "I love Go") {
		t.Errorf("prompt missing the person's text: %s", prompt)
	}
	if captured.ResponseFormat.Type != "json_schema" || captured.ResponseFormat.JSONSchema.Name == "" {
		t.Errorf("response format = %+v, want a named json_schema", captured.ResponseFormat)
	}

	want := []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}
	if len(picks) != 1 || picks[0] != want[0] {
		t.Fatalf("picks = %+v, want %+v", picks, want)
	}
}
