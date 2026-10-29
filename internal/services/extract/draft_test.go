package extract_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

func TestClient_Draft(t *testing.T) {
	t.Parallel()
	dimensions := []dto.DimensionSpec{{Key: dto.DimensionWork}, {Key: dto.DimensionStage}}

	t.Run("sends the phrase and dimension keys, decodes the question", func(t *testing.T) {
		t.Parallel()
		var got openrouter.Request
		client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode request: %v", err)
			}
			replyWith(t, w, `{"question":"Does it offer a 4-day week?","dimension":"work"}`)
		})

		drafted, err := client.Draft(t.Context(), "sk-or-test", "4 day week", dimensions)
		if err != nil {
			t.Fatalf("Draft: %v", err)
		}

		want := dto.DraftedQuestion{Question: "Does it offer a 4-day week?", Dimension: dto.DimensionWork}
		if diff := cmp.Diff(want, drafted); diff != "" {
			t.Errorf("drafted (-want +got):\n%s", diff)
		}
		prompt := got.Messages[0].Content
		for _, part := range []string{"4 day week", "work", "stage"} {
			if !strings.Contains(prompt, part) {
				t.Errorf("prompt missing %q: %s", part, prompt)
			}
		}
	})

	t.Run("malformed reply is an error", func(t *testing.T) {
		t.Parallel()
		client := newClient(t, func(w http.ResponseWriter, _ *http.Request) { replyWith(t, w, `nope`) })
		if _, err := client.Draft(t.Context(), "sk-or-test", "x", dimensions); err == nil {
			t.Fatal("Draft err = nil, want a decode error")
		}
	})
}
