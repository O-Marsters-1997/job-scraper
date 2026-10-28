package lineedit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
)

func TestOpenRouter_Complete(t *testing.T) {
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
	var deltas []string

	c, err := lineedit.NewOpenRouterAt(server.URL, server.Client()).Complete(t.Context(), "sk-or-test",
		lineedit.Prompt{Model: "m/x", System: "sys", User: "usr"}, func(s string) { deltas = append(deltas, s) })

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if diff := cmp.Diff(lineedit.Completion{Text: "Cut p99 latency", Cost: 0.001}, c); diff != "" {
		t.Errorf("Complete() (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"Cut p99 ", "latency"}, deltas); diff != "" {
		t.Errorf("deltas (-want +got):\n%s", diff)
	}
	msgs := got["messages"].([]any)
	if got["model"] != "m/x" || got["stream"] != true || msgs[0].(map[string]any)["content"] != "sys" || msgs[1].(map[string]any)["content"] != "usr" {
		t.Errorf("request = %v, want the prompt streamed to model m/x", got)
	}
	if _, ok := got["response_format"]; ok {
		t.Errorf("request has response_format %v, want plain text", got["response_format"])
	}
}
