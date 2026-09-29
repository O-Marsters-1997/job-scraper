package docedit_test

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
)

type fixtureDoc struct {
	ds   docparse.DocStructure
	text []rune
}

func loadFixture(t *testing.T, name string) fixtureDoc {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docparse", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	ds, err := docparse.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	text := make([]rune, 1)
	collectRuns(tree, &text)
	return fixtureDoc{ds: ds, text: text}
}

func collectRuns(node any, text *[]rune) {
	switch v := node.(type) {
	case map[string]any:
		if run, ok := v["textRun"].(map[string]any); ok {
			start, _ := v["startIndex"].(float64)
			content, _ := run["content"].(string)
			for len(*text) < int(start) {
				*text = append(*text, ' ')
			}
			*text = append((*text)[:int(start)], []rune(content)...)
			return
		}
		for _, child := range v {
			collectRuns(child, text)
		}
	case []any:
		for _, child := range v {
			collectRuns(child, text)
		}
	}
}

func apply(t *testing.T, body []rune, reqs []docedit.Request) string {
	t.Helper()
	doc := append([]rune(nil), body...)
	for _, r := range reqs {
		switch {
		case r.DeleteContentRange != nil:
			rg := r.DeleteContentRange.Range
			doc = append(doc[:rg.StartIndex:rg.StartIndex], doc[rg.EndIndex:]...)
		case r.InsertText != nil:
			i := r.InsertText.Location.Index
			ins := []rune(r.InsertText.Text)
			doc = append(doc[:i:i], append(ins, doc[i:]...)...)
		default:
			t.Fatal("empty request")
		}
	}
	return string(doc)
}

func bullets(texts ...string) []cvedit.Bullet {
	var out []cvedit.Bullet
	for _, s := range texts {
		out = append(out, cvedit.Bullet{Text: s})
	}
	return out
}

func TestRequests(t *testing.T) {
	t.Run("replaces profile, bullets and list skills", func(t *testing.T) {
		doc := loadFixture(t, "profile_skills_list.json")
		positions := docedit.PositionSlots{"p1": {"s0", "s1"}, "p2": {"s2"}}
		profile := "Go engineer."
		edits := cvedit.EditSet{
			Positions: []cvedit.PositionEdit{
				{PositionID: "p1", Bullets: bullets("Cut latency.")},
				{PositionID: "p2", Bullets: bullets("Built ingestion.")},
			},
			Profile: &profile,
			Skills:  []string{"Kubernetes", "Go"},
		}

		reqs, err := docedit.Requests(doc.ds, positions, edits)
		if err != nil {
			t.Fatalf("Requests() error = %v", err)
		}
		got := apply(t, doc.text, reqs)

		want := string(doc.text[:18]) + "Go engineer.\n" +
			string(doc.text[91:145]) + "Cut latency.\n" +
			string(doc.text[267:303]) + "Built ingestion.\n" +
			string(doc.text[357:364]) + "Kubernetes\nGo\n" +
			string(doc.text[389:])
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Requests() applied doc mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("prose skills keep their separator", func(t *testing.T) {
		doc := loadFixture(t, "table_profile_prose_skills.json")
		edits := cvedit.EditSet{Skills: []string{"Docker", "React"}}

		reqs, err := docedit.Requests(doc.ds, nil, edits)
		if err != nil {
			t.Fatalf("Requests() error = %v", err)
		}
		got := apply(t, doc.text, reqs)

		for _, want := range []string{"Docker, React\n", "Technical Skills\n"} {
			if !strings.Contains(got, want) {
				t.Errorf("Requests() applied doc lacks %q:\n%s", want, got)
			}
		}
	})

	t.Run("fewer bullets deletes unused slots", func(t *testing.T) {
		doc := loadFixture(t, "profile_skills_list.json")
		edits := cvedit.EditSet{Positions: []cvedit.PositionEdit{{PositionID: "p1", Bullets: bullets("Only one.")}}}

		reqs, err := docedit.Requests(doc.ds, docedit.PositionSlots{"p1": {"s0", "s1"}}, edits)
		if err != nil {
			t.Fatalf("Requests() error = %v", err)
		}
		got := apply(t, doc.text, reqs)

		want := string(doc.text[:145]) + "Only one.\n" + string(doc.text[267:])
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Requests() applied doc mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("missing profile and skills sections produce no requests", func(t *testing.T) {
		doc := loadFixture(t, "no_profile_skills.json")
		profile := "New profile."
		edits := cvedit.EditSet{Profile: &profile, Skills: []string{"Go"}}

		reqs, err := docedit.Requests(doc.ds, nil, edits)
		if err != nil {
			t.Fatalf("Requests() error = %v", err)
		}
		if len(reqs) != 0 {
			t.Errorf("Requests() = %d requests, want none", len(reqs))
		}
	})

	t.Run("requests descend by document index", func(t *testing.T) {
		doc := loadFixture(t, "profile_skills_list.json")
		profile := "x"
		edits := cvedit.EditSet{
			Positions: []cvedit.PositionEdit{{PositionID: "p1", Bullets: bullets("a", "b")}},
			Profile:   &profile,
			Skills:    []string{"Go"},
		}

		reqs, err := docedit.Requests(doc.ds, docedit.PositionSlots{"p1": {"s0", "s1"}}, edits)
		if err != nil {
			t.Fatalf("Requests() error = %v", err)
		}
		last := math.MaxInt
		for _, r := range reqs {
			idx := 0
			switch {
			case r.DeleteContentRange != nil:
				idx = r.DeleteContentRange.Range.StartIndex
			case r.InsertText != nil:
				idx = r.InsertText.Location.Index
			}
			if idx > last {
				t.Fatalf("Requests() has request at index %d after index %d, want descending", idx, last)
			}
			last = idx
		}
	})
}

func TestRequestsErrors(t *testing.T) {
	tests := []struct {
		name      string
		positions docedit.PositionSlots
		edit      cvedit.PositionEdit
		wantErr   error
	}{
		{
			name:      "more bullets than slots",
			positions: docedit.PositionSlots{"p2": {"s2"}},
			edit:      cvedit.PositionEdit{PositionID: "p2", Bullets: bullets("a", "b")},
			wantErr:   docedit.ErrTooManyBullets,
		},
		{
			name:      "position without slots",
			positions: docedit.PositionSlots{},
			edit:      cvedit.PositionEdit{PositionID: "nope", Bullets: bullets("a")},
			wantErr:   docedit.ErrUnknownPosition,
		},
	}
	doc := loadFixture(t, "profile_skills_list.json")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := docedit.Requests(doc.ds, tt.positions, cvedit.EditSet{Positions: []cvedit.PositionEdit{tt.edit}})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Requests() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
