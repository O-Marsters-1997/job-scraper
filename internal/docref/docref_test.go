package docref_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/docref"
)

func TestParseDocID(t *testing.T) {
	t.Run("extracts the ID", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{"full edit URL", "https://docs.google.com/document/d/DOCID/edit", "DOCID"},
			{"URL with tab param", "https://docs.google.com/document/d/DOCID/edit?tab=t.0", "DOCID"},
			{"path only", "/document/d/DOCID/edit", "DOCID"},
			{"bare ID", "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms", "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := docref.ParseDocID(tt.input)
				if err != nil {
					t.Fatalf("ParseDocID(%q) error = %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("ParseDocID(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})

	t.Run("rejects input with no ID", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{"garbage short string", "not-a-url"},
			{"empty string", ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got, err := docref.ParseDocID(tt.input); err == nil {
					t.Errorf("ParseDocID(%q) = %q, want error", tt.input, got)
				}
			})
		}
	})
}

func FuzzParseDocID(f *testing.F) {
	for _, seed := range []string{
		"https://docs.google.com/document/d/DOCID/edit",
		"https://docs.google.com/document/d/DOCID/edit?tab=t.0",
		"/document/d/DOCID/edit",
		"1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
		"not-a-url",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(_ *testing.T, in string) {
		_, _ = docref.ParseDocID(in)
	})
}
