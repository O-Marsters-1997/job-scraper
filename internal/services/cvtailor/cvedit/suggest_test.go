package cvedit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

func TestSuggest(t *testing.T) {
	t.Run("streams a fit rewrite", func(t *testing.T) {
		var got map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode request: %v", err)
			}
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Cut p99 \"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{\"content\":\"latency\"}}],\"usage\":{\"cost\":0.001}}\n\n" +
				"data: [DONE]\n\n"))
		}))
		t.Cleanup(server.Close)
		client := cvedit.NewClientAt(server.URL, server.Client())
		in := cvedit.SuggestInput{
			Action: "fit", Text: "Cut p99 latency by moving queries to Postgres", MaxChars: 30,
			Achievements: []string{"Cut p99 latency by moving queries to Postgres"},
		}
		var deltas []string

		res, err := client.Suggest(t.Context(), "sk-or-test", in, func(s string) { deltas = append(deltas, s) })

		if err != nil {
			t.Fatalf("Suggest() err = %v", err)
		}
		if diff := cmp.Diff(cvedit.SuggestResult{Text: "Cut p99 latency", Cost: 0.001}, res); diff != "" {
			t.Errorf("Suggest() (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"Cut p99 ", "latency"}, deltas); diff != "" {
			t.Errorf("deltas (-want +got):\n%s", diff)
		}
		if got["model"] != cvedit.SuggestModel || got["stream"] != true {
			t.Errorf("request model/stream = %v/%v, want %s streaming", got["model"], got["stream"], cvedit.SuggestModel)
		}
		if _, ok := got["response_format"]; ok {
			t.Errorf("request has response_format %v, want plain text", got["response_format"])
		}
		msgs := got["messages"].([]any)
		system, user := msgs[0].(map[string]any)["content"].(string), msgs[1].(map[string]any)["content"].(string)
		if !strings.Contains(system, "You rewrite the bullets of an existing CV") {
			t.Errorf("system prompt = %q, want it to include voice.md", system)
		}
		for _, want := range []string{"at most 30 characters", in.Text, in.Achievements[0]} {
			if !strings.Contains(user, want) {
				t.Errorf("user message = %q, want it to contain %q", user, want)
			}
		}
	})

	const original = "Built Kubernetes tooling"
	run := func(t *testing.T, replies ...string) (cvedit.SuggestResult, []string, []string) {
		t.Helper()
		var prompts []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Messages []struct{ Content string }
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode request: %v", err)
			}
			prompts = append(prompts, req.Messages[1].Content)
			text, err := json.Marshal(replies[min(len(prompts), len(replies))-1])
			if err != nil {
				t.Errorf("encode reply: %v", err)
			}
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":" + string(text) + "}}]}\n\ndata: [DONE]\n\n"))
		}))
		t.Cleanup(server.Close)
		client := cvedit.NewClientAt(server.URL, server.Client())
		var deltas []string

		res, err := client.Suggest(t.Context(), "k", cvedit.SuggestInput{Action: "tailor", Quote: "k8s", Text: original}, func(s string) { deltas = append(deltas, s) })

		if err != nil {
			t.Fatalf("Suggest() err = %v", err)
		}
		return res, prompts, deltas
	}

	t.Run("a reply within the rule is not retried", func(t *testing.T) {
		res, prompts, _ := run(t, "Built k8s tooling")

		if res.Text != "Built k8s tooling" || len(prompts) != 1 {
			t.Errorf("Suggest() = %q after %d calls, want the first reply after 1", res.Text, len(prompts))
		}
	})

	t.Run("a reply with more words is retried with the reason", func(t *testing.T) {
		res, prompts, _ := run(t, "Built Kubernetes tooling at scale", "Built k8s tooling")

		if res.Text != "Built k8s tooling" || len(prompts) != 2 {
			t.Fatalf("Suggest() = %q after %d calls, want the second reply after 2", res.Text, len(prompts))
		}
		if strings.Contains(prompts[0], "previous reply was rejected") || !strings.Contains(prompts[1], "more words than the original. Yours had 5 words and 33 characters; use at most 3 words and 24 characters") {
			t.Errorf("prompts = %q, want the reason only on the retry", prompts)
		}
	})

	t.Run("a second bad reply is returned after one retry", func(t *testing.T) {
		res, prompts, _ := run(t, "Built Kubernetes tooling at scale", "Built Kubernetes tooling at huge scale")

		if res.Text != "Built Kubernetes tooling at huge scale" || len(prompts) != 2 {
			t.Errorf("Suggest() = %q after %d calls, want the second reply after 2", res.Text, len(prompts))
		}
	})

	t.Run("streams no deltas", func(t *testing.T) {
		_, _, deltas := run(t, "Built Kubernetes tooling at scale", "Built k8s tooling")

		if len(deltas) != 0 {
			t.Errorf("deltas = %q, want none", deltas)
		}
	})
}

func TestSuggestRejectsUnknownAction(t *testing.T) {
	client := cvedit.NewClientAt("http://unused", http.DefaultClient)

	_, err := client.Suggest(t.Context(), "k", cvedit.SuggestInput{Action: "rewrite", Text: "x"}, func(string) {})

	if err == nil {
		t.Error("Suggest(action rewrite) err = nil, want an error")
	}
}
