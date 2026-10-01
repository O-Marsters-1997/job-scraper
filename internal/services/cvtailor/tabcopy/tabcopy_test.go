package tabcopy_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	src, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func requests(t *testing.T, fixture string) []map[string]map[string]any {
	t.Helper()
	raw, err := tabcopy.Requests(readFixture(t, fixture), "t.1")
	if err != nil {
		t.Fatalf("Requests(%s) error = %v", fixture, err)
	}
	out := make([]map[string]map[string]any, len(raw))
	for i, r := range raw {
		if err := json.Unmarshal(r, &out[i]); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func find(reqs []map[string]map[string]any, kind string, start float64) map[string]any {
	for _, r := range reqs {
		body, ok := r[kind]
		if !ok {
			continue
		}
		rng, _ := body["range"].(map[string]any)
		if rng["startIndex"] == start {
			return body
		}
	}
	return nil
}

func TestRequests(t *testing.T) {
	reqs := requests(t, "cv.json")

	t.Run("inserts the body text without the final newline, then the page setup", func(t *testing.T) {
		if _, ok := reqs[0]["updateDocumentStyle"]; !ok {
			t.Fatalf("first request = %v, want updateDocumentStyle", reqs[0])
		}
		got := reqs[1]["insertText"]["text"]
		if want := "Skills\nGo dev\nOne\nTwo"; got != want {
			t.Errorf("inserted text = %q, want %q", got, want)
		}
		if got := reqs[0]["updateDocumentStyle"]["fields"]; got != "marginBottom,marginLeft,marginRight,marginTop,pageSize" {
			t.Errorf("document style fields = %v", got)
		}
	})

	t.Run("resolves named styles under overrides", func(t *testing.T) {
		got := find(reqs, "updateTextStyle", 1)
		want := map[string]any{
			"range":     map[string]any{"startIndex": 1.0, "endIndex": 8.0, "tabId": "t.1"},
			"textStyle": map[string]any{"bold": true, "fontSize": map[string]any{"magnitude": 14.0, "unit": "PT"}},
			"fields":    "bold,fontSize",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("heading text style (-want +got):\n%s", diff)
		}

		got = find(reqs, "updateTextStyle", 8)
		if diff := cmp.Diff("fontSize,italic", got["fields"]); diff != "" {
			t.Errorf("override text style fields (-want +got):\n%s", diff)
		}

		got = find(reqs, "updateParagraphStyle", 1)
		if diff := cmp.Diff("lineSpacing,spaceAbove", got["fields"]); diff != "" {
			t.Errorf("heading paragraph style fields (-want +got):\n%s", diff)
		}
	})

	t.Run("keeps list indents per nesting level", func(t *testing.T) {
		got := find(reqs, "createParagraphBullets", 15)
		rng := got["range"].(map[string]any)
		if rng["endIndex"] != 23.0 {
			t.Errorf("bullet range = %v, want one range over both items", rng)
		}
		for start, wantIndent := range map[float64]float64{15: 36, 19: 72} {
			ps := find(reqs, "updateParagraphStyle", start)["paragraphStyle"].(map[string]any)
			indent := ps["indentStart"].(map[string]any)["magnitude"]
			if indent != wantIndent {
				t.Errorf("indentStart at %v = %v, want %v", start, indent, wantIndent)
			}
		}
	})
}

func TestRequestsUnsupported(t *testing.T) {
	for _, fixture := range []string{"table.json", "image.json", "columns.json", "header.json", "sections.json"} {
		t.Run(fixture, func(t *testing.T) {
			if _, err := tabcopy.Requests(readFixture(t, fixture), "t.1"); !errors.Is(err, tabcopy.ErrUnsupported) {
				t.Errorf("Requests(%s) error = %v, want ErrUnsupported", fixture, err)
			}
		})
	}
}
