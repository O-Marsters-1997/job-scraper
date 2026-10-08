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
	"text/tabwriter"

	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

//go:embed fixtures/*.json fixtures/suggest/*.json
var fixtureFS embed.FS

const maxRetries = 2

var checkNames = []string{"grounding", "skills", "banned_words", "slot_length", "page_count"}

type Position struct {
	ID           string
	Employer     string
	Title        string
	HeadingIndex int
	Achievements []cvedit.Achievement
}

type Fixture struct {
	Name           string
	Scenario       string
	JobDescription string
	JobTerms       []string
	Positions      []Position
	Structure      docparse.DocStructure
}

func Fixtures() ([]Fixture, error) {
	return loadFixtures("fixtures", func(f *Fixture) *string { return &f.Name })
}

func loadFixtures[T any](dir string, name func(*T) *string) ([]T, error) {
	entries, err := fs.ReadDir(fixtureFS, dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []T
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := fixtureFS.ReadFile(path.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read fixture %s: %w", e.Name(), err)
		}
		var f T
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("fixture %s: %w", e.Name(), err)
		}
		if n := name(&f); *n == "" {
			*n = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, f)
	}
	return out, nil
}

type Editor interface {
	Edit(ctx context.Context, apiKey string, in cvedit.Input) (cvedit.Result, error)
}

type Outcome struct {
	Fixture  string
	Attempts [][]checks.Finding
	Cost     float64

	TermsSupported int
	TermsUsed      int
}

func (o Outcome) Retries() int { return len(o.Attempts) - 1 }

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
		blocks := checks.Blocking(findings)
		if len(blocks) == 0 || len(out.Attempts) > maxRetries {
			out.TermsSupported, out.TermsUsed = termCoverage(f, res.Edits)
			return out, nil
		}
		edits := res.Edits
		in.PriorEdits = &edits
		in.PriorFindings = blocks
	}
}

func termCoverage(f Fixture, edits cvedit.EditSet) (supported, used int) {
	var bank, output strings.Builder
	for _, p := range f.Positions {
		for _, a := range p.Achievements {
			bank.WriteString(a.Text + "\n")
		}
	}
	for _, pe := range edits.Positions {
		for _, b := range pe.Bullets {
			output.WriteString(b.Text + "\n")
		}
	}
	if edits.Profile != nil {
		output.WriteString(*edits.Profile + "\n")
	}
	output.WriteString(strings.Join(cvedit.FlatSkills(edits.Skills), "\n"))
	bankText, outputText := strings.ToLower(bank.String()), strings.ToLower(output.String())
	for _, term := range f.JobTerms {
		term = strings.ToLower(term)
		if !strings.Contains(bankText, term) {
			continue
		}
		supported++
		if strings.Contains(outputText, term) {
			used++
		}
	}
	return supported, used
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
		in.HasSkills, in.BaseSkills = true, cvedit.SkillGroups(s)
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
		d.BaseSkills = cvedit.CheckLines(cvedit.SkillGroups(s))
	}
	d.Skills = cvedit.CheckLines(edits.Skills)
	return d
}

func Report(promptVersion, model string, outcomes []Outcome) string {
	var b strings.Builder
	fmt.Fprintf(&b, "prompt_version=%s model=%s runs=%d\n\n", promptVersion, model, len(outcomes))
	writeCheckTable(&b, outcomes)
	writeTermCoverage(&b, outcomes)
	writeRetrySummary(&b, outcomes)
	return b.String()
}

func writeCheckTable(b *strings.Builder, outcomes []Outcome) {
	var rows strings.Builder
	rows.WriteString("check\tfirst attempt\tafter retries\n")
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
		fmt.Fprintf(&rows, "%s\t%s\t%s\n", name, rate(first, len(outcomes)), rate(final, len(outcomes)))
	}
	tw := tabwriter.NewWriter(b, 0, 4, 2, ' ', 0)
	_, _ = tw.Write([]byte(rows.String()))
	_ = tw.Flush()
}

func writeTermCoverage(b *strings.Builder, outcomes []Outcome) {
	supported, used := 0, 0
	perFixture := map[string][2]int{}
	for _, o := range outcomes {
		supported += o.TermsSupported
		used += o.TermsUsed
		c := perFixture[o.Fixture]
		perFixture[o.Fixture] = [2]int{c[0] + o.TermsUsed, c[1] + o.TermsSupported}
	}
	if supported == 0 {
		return
	}
	fmt.Fprintf(b, "\nterm_coverage %s\n", rate(used, supported))
	for _, name := range slices.Sorted(maps.Keys(perFixture)) {
		if c := perFixture[name]; c[1] > 0 {
			fmt.Fprintf(b, "  %s: %s\n", name, rate(c[0], c[1]))
		}
	}
}

func writeRetrySummary(b *strings.Builder, outcomes []Outcome) {
	retries, cost := 0, 0.0
	perFixture := map[string][]int{}
	for _, o := range outcomes {
		retries += o.Retries()
		cost += o.Cost
		perFixture[o.Fixture] = append(perFixture[o.Fixture], o.Retries())
	}
	b.WriteString("\nretries per run, by fixture\n")
	for _, name := range slices.Sorted(maps.Keys(perFixture)) {
		fmt.Fprintf(b, "%s: %v\n", name, perFixture[name])
	}
	fmt.Fprintf(b, "\ntotal retries=%d cost=$%.4f\n", retries, cost)
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
