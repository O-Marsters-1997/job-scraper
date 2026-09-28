package docref

import (
	"testing"
)

func TestParseDocID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full edit URL",
			input: "https://docs.google.com/document/d/DOCID/edit",
			want:  "DOCID",
		},
		{
			name:  "URL with tab param",
			input: "https://docs.google.com/document/d/DOCID/edit?tab=t.0",
			want:  "DOCID",
		},
		{
			name:  "path only",
			input: "/document/d/DOCID/edit",
			want:  "DOCID",
		},
		{
			name:  "bare ID",
			input: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
			want:  "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDocID(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseDocID_Rejects(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "garbage short string", input: "not-a-url"},
		{name: "empty string", input: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDocID(tc.input)
			if err == nil {
				t.Fatalf("expected error, got %q", got)
			}
		})
	}
}
