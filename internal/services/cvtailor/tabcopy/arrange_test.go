package tabcopy_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

type line struct {
	style, text string
	bullet      bool
}

func h(text string) line { return line{style: "HEADING_2", text: text} }
func b(text string) line { return line{style: "NORMAL_TEXT", text: text, bullet: true} }
func p(text string) line { return line{style: "NORMAL_TEXT", text: text} }

func tabOf(lines ...line) json.RawMessage {
	content := []string{`{"endIndex":1,"sectionBreak":{}}`}
	idx := 1
	for _, l := range lines {
		end := idx + len(l.text) + 1
		bullet := ""
		if l.bullet {
			bullet = `,"bullet":{"listId":"kix.l1"}`
		}
		content = append(content, fmt.Sprintf(
			`{"startIndex":%d,"endIndex":%d,"paragraph":{"elements":[{"startIndex":%d,"endIndex":%d,"textRun":{"content":%q}}],"paragraphStyle":{"namedStyleType":%q}%s}}`,
			idx, end, idx, end, l.text+"\n", l.style, bullet))
		idx = end
	}
	return json.RawMessage(`{"documentTab":{"body":{"content":[` + strings.Join(content, ",") + `]}}}`)
}

func textOf(t *testing.T, tab json.RawMessage) []string {
	t.Helper()
	text, err := docparse.Text(tab)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(text, "\n")
}

func TestArrange(t *testing.T) {
	t.Parallel()
	src := tabOf(
		p("Jane Doe"),
		h("Profile"), p("Engineer."),
		h("Engineer"), h("Acme 2020-2022"), b("a1"), b("a2"), p("Reference available"),
		h("Analyst"), h("Widgets 2018-2020"), b("w1"),
		h("Education"), p("BSc"),
		h("Lead"), h("Globex 2022-now"), b("g1"),
	)
	roleOf := map[string]string{"Acme 2020-2022": "acme", "Widgets 2018-2020": "widgets", "Globex 2022-now": "globex"}

	tests := []struct {
		name  string
		order []string
		want  []string
	}{
		{
			name:  "moves each role block whole into the order, leaving other headings in place",
			order: []string{"globex", "acme", "widgets"},
			want: []string{"Jane Doe", "Profile", "Engineer.",
				"Lead", "Globex 2022-now", "g1",
				"Engineer", "Acme 2020-2022", "a1", "a2", "Reference available",
				"Education", "BSc",
				"Analyst", "Widgets 2018-2020", "w1"},
		},
		{
			name:  "drops a role left out of the order with its block",
			order: []string{"widgets", "globex"},
			want: []string{"Jane Doe", "Profile", "Engineer.",
				"Analyst", "Widgets 2018-2020", "w1",
				"Lead", "Globex 2022-now", "g1",
				"Education", "BSc"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tabcopy.Arrange(src, roleOf, tc.order)
			if err != nil {
				t.Fatalf("Arrange() error = %v", err)
			}
			if diff := cmp.Diff(tc.want, textOf(t, got)); diff != "" {
				t.Errorf("Arrange() text (-want +got):\n%s", diff)
			}
			ds, err := docparse.Parse(got)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range ds.Slots {
				if want := strings.Index(strings.Join(tc.want, "\n"), s.Text); want < 0 {
					t.Errorf("slot %q missing", s.Text)
				}
			}
			reqs, err := tabcopy.Requests(got, "t.1")
			if err != nil {
				t.Fatalf("Requests(Arrange()) error = %v", err)
			}
			var first map[string]map[string]any
			if err := json.Unmarshal(reqs[1], &first); err != nil {
				t.Fatal(err)
			}
			if want := strings.Join(tc.want, "\n"); first["insertText"]["text"] != want {
				t.Errorf("inserted text = %q, want %q", first["insertText"]["text"], want)
			}
		})
	}

	t.Run("reindexes so the moved paragraphs are contiguous", func(t *testing.T) {
		t.Parallel()
		got, err := tabcopy.Arrange(src, roleOf, []string{"globex", "acme", "widgets"})
		if err != nil {
			t.Fatal(err)
		}
		ds, err := docparse.Parse(got)
		if err != nil {
			t.Fatal(err)
		}
		if ds.Slots[0].Text != "g1" || ds.Slots[0].StartIndex != len("Jane Doe\nProfile\nEngineer.\nLead\nGlobex 2022-now\n")+1 {
			t.Errorf("first slot = %+v, want g1 at its new index", ds.Slots[0])
		}
	})

	t.Run("a role mapped to two headings is ambiguous", func(t *testing.T) {
		t.Parallel()
		_, err := tabcopy.Arrange(src, map[string]string{"Acme 2020-2022": "x", "Widgets 2018-2020": "x"}, []string{"x"})
		if err == nil {
			t.Error("Arrange() error = nil, want an error")
		}
	})
}

func TestClear(t *testing.T) {
	t.Parallel()
	reqs, err := tabcopy.Clear(tabOf(h("A"), b("x")), "t.1")
	if err != nil {
		t.Fatal(err)
	}
	var del map[string]map[string]any
	if err := json.Unmarshal(reqs[0], &del); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"startIndex": 1.0, "endIndex": 4.0, "tabId": "t.1"}
	if diff := cmp.Diff(want, del["deleteContentRange"]["range"]); diff != "" {
		t.Errorf("Clear() range (-want +got):\n%s", diff)
	}
}
