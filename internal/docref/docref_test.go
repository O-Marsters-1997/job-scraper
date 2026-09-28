package docref

import (
	"testing"
)

func TestParseDocID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, got string, err error)
	}{
		{
			name:  "full edit URL",
			input: "https://docs.google.com/document/d/DOCID/edit",
			check: wantDocID("DOCID"),
		},
		{
			name:  "URL with tab param",
			input: "https://docs.google.com/document/d/DOCID/edit?tab=t.0",
			check: wantDocID("DOCID"),
		},
		{
			name:  "path only",
			input: "/document/d/DOCID/edit",
			check: wantDocID("DOCID"),
		},
		{
			name:  "bare ID",
			input: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
			check: wantDocID("1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms"),
		},
		{
			name:  "garbage short string",
			input: "not-a-url",
			check: wantParseError,
		},
		{
			name:  "empty string",
			input: "",
			check: wantParseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDocID(tt.input)
			tt.check(t, got, err)
		})
	}
}

func wantDocID(want string) func(t *testing.T, got string, err error) {
	return func(t *testing.T, got string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func wantParseError(t *testing.T, got string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got %q", got)
	}
}
