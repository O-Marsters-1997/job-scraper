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
	inputs := []string{
		"https://www.linkedin.com/jobs/view/1234567890",
		"https://indeed.com/viewjob?jk=abc123",
		"https://boards.greenhouse.io/acme/jobs/123",
		"",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			gotURL, gotType, gotOK := RewriteToATS(input)
			if gotURL != "" || gotType != UnknownHTML || gotOK != false {
				t.Errorf("RewriteToATS(%q) = (%q, %v, %v), want (%q, %v, %v)",
					input, gotURL, gotType, gotOK, "", UnknownHTML, false)
			}
		})
	}
}
