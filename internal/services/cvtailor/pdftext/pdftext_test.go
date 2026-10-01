package pdftext_test

import (
	"os"
	"path/filepath"
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
