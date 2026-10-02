package tabcopy_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

func layoutOf(t *testing.T, fixture string) tabcopy.Document {
	t.Helper()
	got, err := tabcopy.Layout(readFixture(t, fixture))
	if err != nil {
		t.Fatalf("Layout(%s) error = %v", fixture, err)
	}
	return got
}

func TestLayoutResolvesNamedStylesUnderOverrides(t *testing.T) {
	got := layoutOf(t, "cv.json")

	wantPage := dto.LayoutPage{Width: 595, Height: 842, MarginTop: 36, MarginBottom: 36, MarginLeft: 40, MarginRight: 40}
	if diff := cmp.Diff(wantPage, got.Page); diff != "" {
		t.Errorf("Layout().Page (-want +got):\n%s", diff)
	}
	heading, body := got.Blocks[0], got.Blocks[1]
	if heading.StartIndex != 1 || heading.SpaceAbove != 12 || heading.LineSpacing != 115 {
		t.Errorf("heading block = %+v, want start 1, space above 12 and the normal line spacing", heading)
	}
	wantHeading := []dto.LayoutRun{{Text: "Skills", Font: "Arial", Size: 14, Bold: true}}
	if diff := cmp.Diff(wantHeading, heading.Runs); diff != "" {
		t.Errorf("heading runs (-want +got):\n%s", diff)
	}
	wantBody := []dto.LayoutRun{{Text: "Go dev", Font: "Arial", Size: 10, Italic: true}}
	if diff := cmp.Diff(wantBody, body.Runs); diff != "" {
		t.Errorf("body runs (-want +got):\n%s", diff)
	}
}

func TestLayoutIndentsBulletsPerNestingLevel(t *testing.T) {
	got := layoutOf(t, "cv.json")

	for i, want := range []struct{ start, first float64 }{{36, 18}, {72, 54}} {
		b := got.Blocks[2+i]
		if b.Bullet == nil || b.Bullet.Level != i || b.IndentStart != want.start || b.IndentFirstLine != want.first {
			t.Errorf("bullet block %d = %+v, want level %d indents %v", i, b, i, want)
		}
	}
	if got.Blocks[1].Bullet != nil {
		t.Errorf("body block has bullet %+v, want none", got.Blocks[1].Bullet)
	}
}

func TestLayoutReadsBordersTabsAndSpacers(t *testing.T) {
	got := layoutOf(t, "layout.json")

	rule, role, spacer := got.Blocks[0], got.Blocks[1], got.Blocks[2]
	wantRule := &dto.LayoutBorder{Width: 0.75, Color: "#222222", Padding: 1, Dash: "solid"}
	if diff := cmp.Diff(wantRule, rule.BorderBottom); diff != "" {
		t.Errorf("rule border (-want +got):\n%s", diff)
	}
	if rule.BorderTop != nil || rule.Align != "center" {
		t.Errorf("rule block = %+v, want no top border and centred", rule)
	}
	wantTabs := []dto.LayoutTab{{Offset: 523, Alignment: "end"}}
	if diff := cmp.Diff(wantTabs, role.TabStops); diff != "" {
		t.Errorf("role tab stops (-want +got):\n%s", diff)
	}
	wantRuns := []dto.LayoutRun{
		{Text: "Engineer", Font: "Carlito", Size: 11, Bold: true, Color: "#ff0000", Link: "https://example.com"},
		{Text: "\tJan 2024", Font: "Carlito", Size: 11, Underline: true},
	}
	if diff := cmp.Diff(wantRuns, role.Runs); diff != "" {
		t.Errorf("role runs (-want +got):\n%s", diff)
	}
	wantSpacer := []dto.LayoutRun{{Text: "\n", Font: "Carlito", Size: 5}}
	if diff := cmp.Diff(wantSpacer, spacer.Runs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("spacer runs (-want +got):\n%s", diff)
	}
}

func TestLayoutUnsupported(t *testing.T) {
	for _, fixture := range []string{"table.json", "image.json", "columns.json", "header.json"} {
		t.Run(fixture, func(t *testing.T) {
			if _, err := tabcopy.Layout(readFixture(t, fixture)); !errors.Is(err, tabcopy.ErrUnsupported) {
				t.Errorf("Layout(%s) error = %v, want ErrUnsupported", fixture, err)
			}
		})
	}
}
