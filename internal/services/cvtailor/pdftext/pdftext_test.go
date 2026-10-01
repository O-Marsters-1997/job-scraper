package pdftext_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/pdftext"
)

func TestLines(t *testing.T) {
	tests := []struct {
		name string
		file string
		want []string
	}{
		{"single column reads one line per row", "single-column.pdf", []string{
			"Experience", "Built the billing pipeline in Go", "Cut deploy time by half",
			"Education", "BSc Computer Science",
		}},
		{"two columns merge into shared rows", "two-column.pdf", []string{
			"Experience", "Built the billing pipeline in Go Led a team of five engineers",
			"Cut deploy time by half Mentored two junior developers",
		}},
		{"table cells merge into shared rows", "table.pdf", []string{
			"Experience Built the billing pipeline in Go", "Education BSc Computer Science",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			got, err := pdftext.Lines(data)
			if err != nil {
				t.Fatalf("Lines() error = %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Lines() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("not a PDF returns an error", func(t *testing.T) {
		if _, err := pdftext.Lines([]byte("not a pdf")); err == nil {
			t.Error("Lines(garbage) error = nil, want error")
		}
	})
}

func readRuns(t *testing.T, file string) []pdftext.Run {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		t.Fatal(err)
	}
	runs, err := pdftext.Runs(data)
	if err != nil {
		t.Fatalf("Runs(%s) error = %v", file, err)
	}
	return runs
}

func TestRuns(t *testing.T) {
	t.Run("two columns read top to bottom then left to right", func(t *testing.T) {
		var got []string
		for _, r := range readRuns(t, "two-column.pdf") {
			got = append(got, r.Text)
		}
		want := []string{
			"Experience", "Built the billing pipeline in Go", "Led a team of five engineers",
			"Cut deploy time by half", "Mentored two junior developers",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Runs() texts mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("run carries position size and page", func(t *testing.T) {
		got := readRuns(t, "runs-base.pdf")[0]
		if got.Page != 1 || got.Text != "Experience" || got.X != 50 || got.Y != 740 || got.Size != 11 {
			t.Errorf("Runs()[0] = %+v, want page 1 Experience at (50, 740) size 11", got)
		}
	})

	t.Run("not a PDF returns an error", func(t *testing.T) {
		if _, err := pdftext.Runs([]byte("not a pdf")); err == nil {
			t.Error("Runs(garbage) error = nil, want error")
		}
	})
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name     string
		other    string
		wantOK   bool
		wantDiff string
	}{
		{"identical PDFs match", "runs-base.pdf", true, ""},
		{"shift within tolerance matches", "runs-shifted.pdf", true, ""},
		{"reflowed text names the first differing run", "runs-reflowed.pdf", false, `page 1, line 2, "Built the billing pipeline in Go": text reflowed`},
		{"font size change names the run", "runs-resized.pdf", false, `page 1, line 3, "Cut deploy time by half": font size changed from 11 to 12`},
	}
	base := readRuns(t, "runs-base.pdf")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, diff := pdftext.Match(base, readRuns(t, tt.other), 0.5)
			if ok != tt.wantOK || !strings.HasPrefix(diff, tt.wantDiff) {
				t.Errorf("Match() = %v, %q; want %v, prefix %q", ok, diff, tt.wantOK, tt.wantDiff)
			}
		})
	}

	t.Run("shift beyond tolerance fails", func(t *testing.T) {
		if ok, _ := pdftext.Match(base, readRuns(t, "runs-shifted.pdf"), 0.1); ok {
			t.Error("Match(tol 0.1) = true, want false")
		}
	})
}
