package checks_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
)

func parseDraft(in *checks.ParseInput, bullets ...string) checks.Draft {
	slots := make([]checks.Slot, len(bullets))
	for i, b := range bullets {
		slots[i] = checks.Slot{Text: b}
	}
	return checks.Draft{Positions: []checks.Position{{Bullets: slots}}, Parse: in}
}

func TestParse(t *testing.T) {
	bullets := []string{"Built the billing pipeline in Go", "Led a team of five engineers", "Cut deploy time by half"}
	tests := []struct {
		name  string
		draft checks.Draft
		want  int
	}{
		{"nil parse emits nothing", parseDraft(nil, bullets...), 0},
		{"clean single column", parseDraft(&checks.ParseInput{
			Lines:    []string{"Experience", "● Built the billing pipeline in Go", "● Led a team of five engineers", "● Cut deploy time by half"},
			Headings: []string{"Experience"},
		}, bullets...), 0},
		{"ligatures and bullet glyphs fold", parseDraft(&checks.ParseInput{
			Lines: []string{"• Cut deploy time by half"},
		}, "Cut deploy time by half"), 0},
		{"columns interleave bullets", parseDraft(&checks.ParseInput{
			Lines: []string{"Built the billing pipeline in Go Cut deploy time by half", "Led a team of five engineers"},
		}, bullets...), 1},
		{"heading beside other text", parseDraft(&checks.ParseInput{
			Lines:    []string{"Experience Built the billing pipeline in Go"},
			Headings: []string{"Experience"},
		}), 1},
		{"missing bullets are not judged", parseDraft(&checks.ParseInput{Lines: []string{"nothing here"}}, bullets...), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checks.Parse(tt.draft)
			if len(got) != tt.want {
				t.Fatalf("Parse() = %v, want %d findings", got, tt.want)
			}
			for _, f := range got {
				if diff := cmp.Diff(checks.Info, f.Severity); diff != "" || f.Check != "parse" {
					t.Errorf("finding = %+v, want parse/info", f)
				}
			}
		})
	}
}
