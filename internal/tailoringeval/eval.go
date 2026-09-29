// Package tailoringeval runs cvedit and checks over fixtures against a real
// model and reports check pass rates and retry counts per prompt version.
package tailoringeval

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/services/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvedit"
)

//go:embed fixtures/*.json
var fixtureFS embed.FS

const maxRetries = 2

var checkNames = []string{"grounding", "skills", "banned_words", "slot_length", "page_count"}

// Position is one Bank Position mapped to the base CV heading it sits under.
type Position struct {
	ID           string
	Employer     string
	Title        string
	HeadingIndex int
	Achievements []cvedit.Achievement
}

// Fixture is a Bank, a job description and a parsed base CV, so no Google
// call is needed.
type Fixture struct {
	Name           string
	Scenario       string
	JobDescription string
	Positions      []Position
	Structure      docparse.DocStructure
}

// Fixtures loads the embedded fixture set.
func Fixtures() ([]Fixture, error) {
	entries, err := fs.ReadDir(fixtureFS, "fixtures")
	if err != nil {
		return nil, err
	}
	var out []Fixture
	for _, e := range entries {
		raw, err := fixtureFS.ReadFile(path.Join("fixtures", e.Name()))
		if err != nil {
			return nil, err
		}
		var f Fixture
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("fixture %s: %w", e.Name(), err)
		}
		if f.Name == "" {
			f.Name = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, f)
	}
	return out, nil
}

// Editor is the slice of cvedit.Client the evaluation uses.
type Editor interface {
	Edit(ctx context.Context, apiKey string, in cvedit.Input) (cvedit.Result, error)
}

// Outcome is one fixture run: the findings after each attempt.
type Outcome struct {
	Fixture  string
	Attempts [][]checks.Finding
	Cost     float64
}

// Retries is how many attempts followed the first.
func (o Outcome) Retries() int { return len(o.Attempts) - 1 }

// Run edits the fixture, retrying on block findings up to maxRetries times.
func Run(ctx context.Context, ed Editor, apiKey string, f Fixture) (Outcome, error) {
	out := Outcome{Fixture: f.Name}
	in := input(f)
	for {
		res, err := ed.Edit(ctx, apiKey, in)
		out.Cost += res.Cost
		if err != nil {
			return out, fmt.Errorf("fixture %s: %w", f.Name, err)
		}
		findings := checks.Run(draft(f, res.Edits))
		out.Attempts = append(out.Attempts, findings)
		blocks := blocking(findings)
		if len(blocks) == 0 || len(out.Attempts) > maxRetries {
			return out, nil
		}
		edits := res.Edits
		in.PriorEdits = &edits
		in.PriorFindings = blocks
	}
}

func blocking(findings []checks.Finding) []cvedit.Finding {
	var out []cvedit.Finding
	for _, f := range findings {
		if f.Severity == checks.Block {
			out = append(out, cvedit.Finding{Check: f.Check, SlotID: f.SlotID, Message: f.Message})
		}
	}
	return out
}

func slotsUnder(f Fixture, p Position) []docparse.Slot {
	var out []docparse.Slot
	for _, s := range f.Structure.Slots {
		if s.HeadingIndex == p.HeadingIndex {
			out = append(out, s)
		}
	}
	return out
}

func input(f Fixture) cvedit.Input {
	in := cvedit.Input{JobDescription: f.JobDescription}
	for _, p := range f.Positions {
		cp := cvedit.Position{ID: p.ID, Employer: p.Employer, Title: p.Title, Achievements: p.Achievements}
		for _, s := range slotsUnder(f, p) {
			cp.SlotTexts = append(cp.SlotTexts, s.Text)
		}
		in.Positions = append(in.Positions, cp)
	}
	if p := f.Structure.Profile; p != nil {
		in.HasProfile, in.BaseProfile = true, p.Text
	}
	if s := f.Structure.Skills; s != nil {
		in.HasSkills, in.BaseSkills = true, s.Items
	}
	return in
}

func draft(f Fixture, edits cvedit.EditSet) checks.Draft {
	d := checks.Draft{BasePages: 1}
	byID := map[string]cvedit.PositionEdit{}
	for _, pe := range edits.Positions {
		byID[pe.PositionID] = pe
	}
	for _, p := range f.Positions {
		texts := map[string]string{}
		cp := checks.Position{ID: p.ID}
		for _, a := range p.Achievements {
			texts[a.ID] = a.Text
			cp.Achievements = append(cp.Achievements, a.Text)
			d.Bank = append(d.Bank, a.Text)
		}
		slots := slotsUnder(f, p)
		for i, b := range byID[p.ID].Bullets {
			slot := checks.Slot{ID: fmt.Sprintf("%s-extra%d", p.ID, i), Text: b.Text}
			if i < len(slots) {
				slot.ID, slot.BaseText = slots[i].ID, slots[i].Text
			}
			for _, id := range b.AchievementIDs {
				if t, ok := texts[id]; ok {
					slot.Cited = append(slot.Cited, t)
				}
			}
			cp.Bullets = append(cp.Bullets, slot)
		}
		d.Positions = append(d.Positions, cp)
	}
	if p := f.Structure.Profile; p != nil && edits.Profile != nil {
		d.Profile = &checks.Slot{ID: p.ID, Text: *edits.Profile, BaseText: p.Text}
	}
	if s := f.Structure.Skills; s != nil {
		d.BaseSkills = s.Items
	}
	d.Skills, d.JobSkills = edits.Skills, edits.JobSkills
	return d
}

// Report writes per-check pass rates and retry counts for one prompt version.
// A run passes a check when no block finding from it remains, either on the
// first attempt or after retries.
func Report(w io.Writer, promptVersion, model string, outcomes []Outcome) {
	_, _ = fmt.Fprintf(w, "prompt_version=%s model=%s runs=%d\n\n", promptVersion, model, len(outcomes))
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "check\tfirst attempt\tafter retries")
	for _, name := range checkNames {
		first, final := 0, 0
		for _, o := range outcomes {
			if !hasBlock(o.Attempts[0], name) {
				first++
			}
			if !hasBlock(o.Attempts[len(o.Attempts)-1], name) {
				final++
			}
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", name, rate(first, len(outcomes)), rate(final, len(outcomes)))
	}
	_ = tw.Flush()

	retries, cost := 0, 0.0
	_, _ = fmt.Fprintln(w, "\nretries per run, by fixture")
	names := make([]string, 0, len(outcomes))
	perFixture := map[string][]int{}
	for _, o := range outcomes {
		retries += o.Retries()
		cost += o.Cost
		if _, ok := perFixture[o.Fixture]; !ok {
			names = append(names, o.Fixture)
		}
		perFixture[o.Fixture] = append(perFixture[o.Fixture], o.Retries())
	}
	slices.Sort(names)
	for _, n := range names {
		_, _ = fmt.Fprintf(w, "%s: %v\n", n, perFixture[n])
	}
	_, _ = fmt.Fprintf(w, "\ntotal retries=%d cost=$%.4f\n", retries, cost)
}

func hasBlock(findings []checks.Finding, check string) bool {
	return slices.ContainsFunc(findings, func(f checks.Finding) bool {
		return f.Check == check && f.Severity == checks.Block
	})
}

func rate(n, total int) string {
	if total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%d/%d (%.0f%%)", n, total, 100*float64(n)/float64(total))
}
