package linefit_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/linefit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

func TestLines(t *testing.T) {
	t.Parallel()

	t.Run("matches the browser on every dumped row", func(t *testing.T) {
		t.Parallel()
		data, err := os.ReadFile("testdata/browser.json")
		if err != nil {
			t.Fatal(err)
		}
		var rows []struct {
			Font         string  `json:"font"`
			Size         float64 `json:"size"`
			Bold         bool    `json:"bold"`
			Italic       bool    `json:"italic"`
			WidthPt      float64 `json:"widthPt"`
			Text         string  `json:"text"`
			BrowserLines int     `json:"browserLines"`
		}
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) == 0 {
			t.Fatal("browser dump has no rows")
		}
		for _, r := range rows {
			s := linefit.Style{Font: r.Font, Size: r.Size, Bold: r.Bold, Italic: r.Italic}
			if got := linefit.Lines(r.Text, s, r.WidthPt); got != r.BrowserLines {
				t.Errorf("Lines(%q, %+v, %v) = %d, want %d", r.Text, s, r.WidthPt, got, r.BrowserLines)
			}
		}
	})

	t.Run("unknown families fall back as docFonts.ts does", func(t *testing.T) {
		t.Parallel()
		text := strings.Repeat("Wide MMMM wiggly illicit words ", 8)
		width := 0.0
		lines := func(font string) int {
			return linefit.Lines(text, linefit.Style{Font: font, Size: 11}, width)
		}
		for width = 100; lines("Arial") == lines("Times New Roman"); width++ {
			if width > 400 {
				t.Fatal("no width separates Arial from Times New Roman, so the fallbacks cannot be told apart")
			}
		}
		for font, like := range map[string]string{
			"Georgia":       "Times New Roman",
			"Garamond":      "Times New Roman",
			"Verdana":       "Arial",
			"Comic Sans MS": "Arial",
			"DejaVu Serif":  "Times New Roman",
			"Open Sans":     "Arial",
			"Sans Serif":    "Arial",
			"  arial ":      "Arial",
		} {
			if got, want := lines(font), lines(like); got != want {
				t.Errorf("Lines with font %q = %d, want %d (as %q)", font, got, want, like)
			}
		}
	})

	t.Run("breaks an over-long token by character", func(t *testing.T) {
		t.Parallel()
		got := linefit.Lines(strings.Repeat("W", 200), linefit.Style{Font: "Arial", Size: 11}, 100)
		if got < 2 {
			t.Errorf("Lines(200 W) = %d, want it split over several lines", got)
		}
	})

	t.Run("empty text is one line", func(t *testing.T) {
		t.Parallel()
		if got := linefit.Lines("", linefit.Style{Font: "Arial", Size: 11}, 100); got != 1 {
			t.Errorf("Lines(\"\") = %d, want 1", got)
		}
	})
}

func TestWidth(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("../tabcopy/testdata/cv.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := tabcopy.Layout(src)
	if err != nil {
		t.Fatalf("Layout() error = %v", err)
	}
	for i, want := range []float64{595 - 40 - 40 - 36, 595 - 40 - 40 - 72} {
		if got := linefit.Width(doc.Page, doc.Blocks[2+i].LayoutBlock); got != want {
			t.Errorf("Width(bullet level %d) = %v, want %v", i, got, want)
		}
	}
	if got := linefit.Width(dto.LayoutPage{Width: 612, MarginLeft: 72, MarginRight: 72}, dto.LayoutBlock{}); got != 468 {
		t.Errorf("Width(no indent) = %v, want 468", got)
	}
}
