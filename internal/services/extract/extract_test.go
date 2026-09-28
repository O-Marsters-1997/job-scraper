package extract_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/extract"
)

type wireChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type wireResponseFormat struct {
	Type       string `json:"type"`
	JSONSchema struct {
		Name string `json:"name"`
	} `json:"json_schema"`
}

type wireChatRequest struct {
	Model          string             `json:"model"`
	Messages       []wireChatMessage  `json:"messages"`
	ResponseFormat wireResponseFormat `json:"response_format"`
}

type wireExtractedPick struct {
	OptionID string `json:"optionId"`
	Stance   string `json:"stance"`
}

type wireExtractionResult struct {
	Picks []wireExtractedPick `json:"picks"`
}

type wireChatResponse struct {
	Choices []struct {
		Message wireChatMessage `json:"message"`
	} `json:"choices"`
}

func TestClient_Extract_SendsPromptAndBank_DecodesPicks(t *testing.T) {
	var captured wireChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-or-test" {
			t.Errorf("Authorization header = %q, want Bearer sk-or-test", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		content, err := json.Marshal(wireExtractionResult{
			Picks: []wireExtractedPick{{OptionID: "tech:go", Stance: "nice"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		resp := wireChatResponse{Choices: []struct {
			Message wireChatMessage `json:"message"`
		}{{Message: wireChatMessage{Role: "assistant", Content: string(content)}}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := extract.NewClientAt(server.URL, server.Client())

	options := []dto.ScoringOption{{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go"}}
	dimensions := []dto.DimensionSpec{{Key: dto.DimensionTech, Kind: "pair", Stances: []string{"nice", "avoid"}}}

	picks, err := client.Extract(context.Background(), "sk-or-test", "I love Go", options, dimensions)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if captured.Model != extract.Model {
		t.Errorf("model = %q, want %q", captured.Model, extract.Model)
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

func TestClient_Extract_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := extract.NewClientAt(server.URL, server.Client())

	_, err := client.Extract(context.Background(), "sk-or-test", "I love Go", nil, nil)
	if err == nil {
		t.Fatal("Extract: want error, got nil")
	}
	if want := fmt.Sprintf("status %d", http.StatusInternalServerError); !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to mention %q", err.Error(), want)
	}
}
