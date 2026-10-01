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
}

func TestSuggestRejectsUnknownAction(t *testing.T) {
	client := cvedit.NewClientAt("http://unused", http.DefaultClient)

	_, err := client.Suggest(t.Context(), "k", cvedit.SuggestInput{Action: "rewrite", Text: "x"}, func(string) {})

	if err == nil {
		t.Error("Suggest(action rewrite) err = nil, want an error")
	}
}
