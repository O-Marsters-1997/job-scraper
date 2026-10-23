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

func TestClient_Insights(t *testing.T) {
	t.Parallel()
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		content, _ := json.Marshal(`{"lines":[{"slotId":"s1","source":"jd","quote":"Go engineer","why":"Shows Go work"}]}`)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + string(content) + `}}],"usage":{"cost":0.02}}`))
	}))
	t.Cleanup(server.Close)
	client := cvedit.NewClientAt(server.URL, server.Client())
	lean := 0.5
	in := cvedit.InsightsInput{
		Bullets:        []cvedit.InsightBullet{{SlotID: "s1", Text: "Built Go services", Lean: &lean}},
		JobDescription: "We need a Go engineer.",
		JobNote:        "Likes mentoring",
		JobComments:    []string{"Remote first"},
	}

	res, err := client.Insights(t.Context(), "sk-or-test", in)

	if err != nil {
		t.Fatalf("Insights() err = %v", err)
	}
	want := cvedit.InsightsResult{Lines: []cvedit.InsightLine{{SlotID: "s1", Source: "jd", Quote: "Go engineer", Why: "Shows Go work"}}, Cost: 0.02}
	if diff := cmp.Diff(want, res); diff != "" {
		t.Errorf("Insights() (-want +got):\n%s", diff)
	}
	if got["model"] != cvedit.Model {
		t.Errorf("request model = %v, want %s", got["model"], cvedit.Model)
	}
	msgs, _ := got["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("messages = %v, want system and user", msgs)
	}
	last, _ := msgs[1].(map[string]any)
	user, _ := last["content"].(string)
	for _, part := range []string{"[s1] (lean +50%) Built Go services", "We need a Go engineer.", "Likes mentoring", "Remote first"} {
		if !strings.Contains(user, part) {
			t.Errorf("user message = %q, want it to contain %q", user, part)
		}
	}
}
