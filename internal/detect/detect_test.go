package detect

import (
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		url  string
		want ATSType
	}{
		{"https://boards.greenhouse.io/acme/jobs/123", Greenhouse},
		{"https://boards-api.greenhouse.io/v1/boards/acme/jobs/456", Greenhouse},
		{"https://jobs.lever.co/acme/job-slug", Lever},
		{"https://jobs.ashbyhq.com/acme/role-slug", Ashby},
		{"https://apply.workable.com/acme/j/ABC123/", Workable},
		{"https://acme.recruitee.com/o/software-engineer", Recruitee},
		{"https://acme.personio.de/job/software-engineer-123", Personio},
		{"https://www.linkedin.com/jobs/view/1234567890", Aggregator},
		{"https://indeed.com/viewjob?jk=abc123", Aggregator},
		{"https://workinstartups.com/job-board/job/12345/software-engineer", UnknownHTML},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := Detect(tt.url)
			if got != tt.want {
				t.Errorf("Detect(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestRewriteToATS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantURL  string
		wantType ATSType
		wantOK   bool
	}{
		{
			name:     "linkedin with greenhouse externalUrl",
			input:    "https://www.linkedin.com/jobs/view/123?externalUrl=https%3A%2F%2Fboards.greenhouse.io%2Facme%2Fjobs%2F456",
			wantURL:  "https://boards.greenhouse.io/acme/jobs/456",
			wantType: Greenhouse,
			wantOK:   true,
		},
		{
			name:     "linkedin without externalUrl",
			input:    "https://www.linkedin.com/jobs/view/1234567890",
			wantURL:  "",
			wantType: UnknownHTML,
			wantOK:   false,
		},
		{
			name:     "indeed with lever url param",
			input:    "https://indeed.com/viewjob?url=https%3A%2F%2Fjobs.lever.co%2Facme%2Fabc-123",
			wantURL:  "https://jobs.lever.co/acme/abc-123",
			wantType: Lever,
			wantOK:   true,
		},
		{
			name:     "indeed without url param",
			input:    "https://indeed.com/viewjob?jk=abc123",
			wantURL:  "",
			wantType: UnknownHTML,
			wantOK:   false,
		},
		{
			name:     "non-aggregator URL",
			input:    "https://boards.greenhouse.io/acme/jobs/123",
			wantURL:  "",
			wantType: UnknownHTML,
			wantOK:   false,
		},
		{
			name:     "empty string",
			input:    "",
			wantURL:  "",
			wantType: UnknownHTML,
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotType, gotOK := RewriteToATS(tt.input)
			if gotURL != tt.wantURL || gotType != tt.wantType || gotOK != tt.wantOK {
				t.Errorf("RewriteToATS(%q) = (%q, %v, %v), want (%q, %v, %v)",
					tt.input, gotURL, gotType, gotOK, tt.wantURL, tt.wantType, tt.wantOK)
			}
		})
	}
}
