package docedit_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/docedit"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/services/cvedit"
)

type fixtureDoc struct {
	ds   docparse.DocStructure
	text []rune
}

func loadFixture(t *testing.T, name string) fixtureDoc {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "docparse", "testdata", name))
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

func TestRequests_ListSkillsAndProfile(t *testing.T) {
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
		t.Fatal(err)
	}
	got := apply(t, doc.text, reqs)

	want := string(doc.text[:18]) + "Go engineer.\n" +
		string(doc.text[91:145]) + "Cut latency.\n" +
		string(doc.text[267:303]) + "Built ingestion.\n" +
		string(doc.text[357:364]) + "Kubernetes\nGo\n" +
		string(doc.text[389:])
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("edited doc mismatch (-want +got):\n%s", diff)
	}
}

func TestRequests_ProseSkillsKeepSeparator(t *testing.T) {
	doc := loadFixture(t, "table_profile_prose_skills.json")
	edits := cvedit.EditSet{Skills: []string{"Docker", "React"}}

	reqs, err := docedit.Requests(doc.ds, nil, edits)
	if err != nil {
		t.Fatal(err)
	}
	got := apply(t, doc.text, reqs)

	if !strings.Contains(got, "Docker, React\n") {
		t.Errorf("skills not joined with %q separator:\n%s", ", ", got)
	}
	if !strings.Contains(got, "Technical Skills\n") {
		t.Errorf("heading lost:\n%s", got)
	}
}

func TestRequests_DescendingOrder(t *testing.T) {
	doc := loadFixture(t, "profile_skills_list.json")
	profile := "x"
	edits := cvedit.EditSet{
		Positions: []cvedit.PositionEdit{{PositionID: "p1", Bullets: bullets("a", "b")}},
		Profile:   &profile,
		Skills:    []string{"Go"},
	}
	reqs, err := docedit.Requests(doc.ds, docedit.PositionSlots{"p1": {"s0", "s1"}}, edits)
	if err != nil {
		t.Fatal(err)
	}
	last := int(^uint(0) >> 1)
	for _, r := range reqs {
		idx := 0
		switch {
		case r.DeleteContentRange != nil:
			idx = r.DeleteContentRange.Range.StartIndex
		case r.InsertText != nil:
			idx = r.InsertText.Location.Index
		}
		if idx > last {
			t.Fatalf("request at index %d follows lower index %d", idx, last)
		}
		last = idx
	}
}

func TestRequests_FewerBulletsDeletesUnusedSlots(t *testing.T) {
	doc := loadFixture(t, "profile_skills_list.json")
	edits := cvedit.EditSet{Positions: []cvedit.PositionEdit{{PositionID: "p1", Bullets: bullets("Only one.")}}}

	reqs, err := docedit.Requests(doc.ds, docedit.PositionSlots{"p1": {"s0", "s1"}}, edits)
	if err != nil {
		t.Fatal(err)
	}
	got := apply(t, doc.text, reqs)

	want := string(doc.text[:145]) + "Only one.\n" + string(doc.text[267:])
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("edited doc mismatch (-want +got):\n%s", diff)
	}
}

func TestRequests_MoreBulletsRejected(t *testing.T) {
	doc := loadFixture(t, "profile_skills_list.json")
	edits := cvedit.EditSet{Positions: []cvedit.PositionEdit{{PositionID: "p2", Bullets: bullets("a", "b")}}}

	_, err := docedit.Requests(doc.ds, docedit.PositionSlots{"p2": {"s2"}}, edits)
	if !errors.Is(err, docedit.ErrTooManyBullets) {
		t.Errorf("Requests() error = %v, want ErrTooManyBullets", err)
	}
}

func TestRequests_UnknownPositionRejected(t *testing.T) {
	doc := loadFixture(t, "profile_skills_list.json")
	edits := cvedit.EditSet{Positions: []cvedit.PositionEdit{{PositionID: "nope", Bullets: bullets("a")}}}

	_, err := docedit.Requests(doc.ds, docedit.PositionSlots{}, edits)
	if !errors.Is(err, docedit.ErrUnknownPosition) {
		t.Errorf("Requests() error = %v, want ErrUnknownPosition", err)
	}
}

func TestRequests_SkipsMissingProfileAndSkills(t *testing.T) {
	doc := loadFixture(t, "no_profile_skills.json")
	profile := "New profile."
	edits := cvedit.EditSet{Profile: &profile, Skills: []string{"Go"}}

	reqs, err := docedit.Requests(doc.ds, nil, edits)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 0 {
		t.Errorf("Requests() = %d requests, want none", len(reqs))
	}
}
