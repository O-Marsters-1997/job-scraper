package eval

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

//go:embed fixtures/suggest/*.json
var suggestFS embed.FS

// SuggestFixture is one inline-edit request: a line, the Achievements it may
// draw on and the action asked of the model.
type SuggestFixture struct {
	Name         string
	Scenario     string
	Text         string
	Achievements []string
	Action       string
	Prompt       string
	MaxChars     int
}

func SuggestFixtures() ([]SuggestFixture, error) {
	entries, err := fs.ReadDir(suggestFS, "fixtures/suggest")
	if err != nil {
		return nil, fmt.Errorf("read suggest fixtures: %w", err)
	}
	var out []SuggestFixture
	for _, e := range entries {
		raw, err := suggestFS.ReadFile(path.Join("fixtures/suggest", e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read fixture %s: %w", e.Name(), err)
		}
		var f SuggestFixture
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("suggest fixture %s: %w", e.Name(), err)
		}
		if f.Name == "" {
			f.Name = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, f)
	}
	return out, nil
}

type Suggester interface {
	Suggest(ctx context.Context, apiKey string, in cvedit.SuggestInput, onDelta func(string)) (cvedit.SuggestResult, error)
}

type SuggestOutcome struct {
	Fixture   string
	Action    string
	Output    string
	Findings  []checks.Finding
	OverLimit bool
	Cost      float64
}

// Passed reports whether the suggestion is grounded, free of banned words and
// within the length its action allows.
func (o SuggestOutcome) Passed() bool {
	return len(checks.Blocking(o.Findings)) == 0 && !o.OverLimit
}

func RunSuggest(ctx context.Context, s Suggester, apiKey string, f SuggestFixture) (SuggestOutcome, error) {
	res, err := s.Suggest(ctx, apiKey, cvedit.SuggestInput{
		Action: f.Action, Prompt: f.Prompt, Text: f.Text, MaxChars: f.MaxChars, Achievements: f.Achievements,
	}, func(string) {})
	out := SuggestOutcome{Fixture: f.Name, Action: f.Action, Output: res.Text, Cost: res.Cost}
	if err != nil {
		return out, fmt.Errorf("suggest fixture %s: %w", f.Name, err)
	}
	slot := checks.Slot{ID: "s0", Text: res.Text, BaseText: f.Text, Cited: f.Achievements}
	d := checks.Draft{
		Positions: []checks.Position{{ID: "p1", Achievements: f.Achievements, Bullets: []checks.Slot{slot}}},
		Bank:      f.Achievements,
	}
	out.Findings = append(checks.Grounding(d), checks.BannedWords(d)...)
	length := utf8.RuneCountInString(res.Text)
	switch f.Action {
	case "fit":
		out.OverLimit = length > f.MaxChars
	case "tighten":
		out.OverLimit = length > utf8.RuneCountInString(f.Text)
	}
	return out, nil
}

func SuggestReport(promptVersion, model string, outcomes []SuggestOutcome) string {
	var b strings.Builder
	fmt.Fprintf(&b, "suggest prompt_version=%s model=%s runs=%d\n\n", promptVersion, model, len(outcomes))
	passed, cost := 0, 0.0
	perFixture := map[string][2]int{}
	for _, o := range outcomes {
		cost += o.Cost
		c := perFixture[o.Fixture]
		c[1]++
		if o.Passed() {
			passed++
			c[0]++
		}
		perFixture[o.Fixture] = c
	}
	fmt.Fprintf(&b, "pass %s\n", rate(passed, len(outcomes)))
	for _, name := range slices.Sorted(maps.Keys(perFixture)) {
		c := perFixture[name]
		fmt.Fprintf(&b, "%s: %s\n", name, rate(c[0], c[1]))
	}
	for _, o := range outcomes {
		if o.Passed() {
			continue
		}
		fmt.Fprintf(&b, "\nFAIL %s: %q\n", o.Fixture, o.Output)
		for _, f := range checks.Blocking(o.Findings) {
			fmt.Fprintf(&b, "  [%s] %s\n", f.Check, f.Message)
		}
		if o.OverLimit {
			b.WriteString("  over the length limit\n")
		}
	}
	fmt.Fprintf(&b, "\ntotal cost=$%.4f\n", cost)
	return b.String()
}
