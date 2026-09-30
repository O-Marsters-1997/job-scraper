package extract_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
	"github.com/ollymarsters/job-scraper/internal/services/extract"
)

func newClient(t *testing.T, h http.HandlerFunc) *extract.Client {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return extract.NewClientAt(server.URL, server.Client())
}

func replyWith(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	body := map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content}}}}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Errorf("encode reply: %v", err)
	}
}

func TestExtract(t *testing.T) {
	options := []dto.ScoringOption{{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go"}}
	dimensions := []dto.DimensionSpec{{Key: dto.DimensionTech, Kind: "pair", Stances: []string{"nice", "avoid"}}}

	t.Run("sends prompt and bank, decodes picks", func(t *testing.T) {
		var got openrouter.Request
		var auth string
		client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
			auth = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			replyWith(t, w, `{"picks":[{"optionId":"tech:go","stance":"nice"}]}`)
		})

		picks, err := client.Extract(context.Background(), "sk-or-test", "I love Go", options, dimensions)
		if err != nil {
			t.Fatalf("Extract: %v", err)
		}

		want := []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "text"}}
		if diff := cmp.Diff(want, picks); diff != "" {
			t.Errorf("picks (-want +got):\n%s", diff)
		}
		if auth != "Bearer sk-or-test" {
			t.Errorf("Authorization = %q, want Bearer sk-or-test", auth)
		}
		if got.Model != extract.Model {
			t.Errorf("model = %q, want %q", got.Model, extract.Model)
		}
		if len(got.Messages) != 1 {
			t.Fatalf("messages = %d, want 1", len(got.Messages))
		}
		prompt := got.Messages[0].Content
		for _, part := range []string{"tech:go", "Go", "nice/avoid", "I love Go"} {
			if !strings.Contains(prompt, part) {
				t.Errorf("prompt missing %q: %s", part, prompt)
			}
		}
		if got.ResponseFormat.Type != "json_schema" || got.ResponseFormat.JSONSchema.Name == "" {
			t.Errorf("response format = %+v, want a named json_schema", got.ResponseFormat)
		}
	})

	t.Run("upstream status is a StatusError", func(t *testing.T) {
		client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		_, err := client.Extract(context.Background(), "sk-or-test", "I love Go", nil, nil)
		var statusErr *openrouter.StatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("Extract error = %v, want *openrouter.StatusError", err)
		}
		if statusErr.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.Code, http.StatusInternalServerError)
		}
	})
}
