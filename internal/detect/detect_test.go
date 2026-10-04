package detect_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/detect"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want detect.ATSType
	}{
		{"greenhouse board", "https://boards.greenhouse.io/acme/jobs/123", detect.Greenhouse},
		{"greenhouse api", "https://boards-api.greenhouse.io/v1/boards/acme/jobs/456", detect.Greenhouse},
		{"lever", "https://jobs.lever.co/acme/job-slug", detect.Lever},
		{"ashby", "https://jobs.ashbyhq.com/acme/role-slug", detect.Ashby},
		{"workable", "https://apply.workable.com/acme/j/ABC123/", detect.Workable},
		{"recruitee", "https://acme.recruitee.com/o/software-engineer", detect.Recruitee},
		{"personio", "https://acme.personio.de/job/software-engineer-123", detect.Personio},
		{"pinpoint job", "https://acme.pinpointhq.com/en/postings/ce6c9e5c-a2d3", detect.Pinpoint},
		{"pinpoint board", "https://acme.pinpointhq.com/", detect.Pinpoint},
		{"pinpoint with port", "https://acme.pinpointhq.com:443/", detect.Pinpoint},
		{"teamtailor job", "https://acme.teamtailor.com/jobs/123-engineer", detect.Teamtailor},
		{"teamtailor board", "https://acme.teamtailor.com/", detect.Teamtailor},
		{"hibob job", "https://acme.careers.hibob.com/jobs/abc-123", detect.HiBob},
		{"hibob board", "https://acme.careers.hibob.com/jobs", detect.HiBob},
		{"smartrecruiters job", "https://jobs.smartrecruiters.com/Wise/744000153370939-card-disputes-specialist", detect.SmartRecruiters},
		{"smartrecruiters careers host", "https://careers.smartrecruiters.com/Wise", detect.SmartRecruiters},
		{"linkedin is an aggregator", "https://www.linkedin.com/jobs/view/1234567890", detect.Aggregator},
		{"indeed is an aggregator", "https://indeed.com/viewjob?jk=abc123", detect.Aggregator},
		{"unknown host", "https://workinstartups.com/job-board/job/12345/software-engineer", detect.UnknownHTML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detect.Detect(tt.url)
			if got != tt.want {
				t.Errorf("Detect(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestResolveBoard(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantSource string
		wantToken  string
		wantOK     bool
	}{
		{"greenhouse board", "https://boards.greenhouse.io/acmecorp", "greenhouse", "acmecorp", true},
		{"greenhouse job url", "https://boards.greenhouse.io/acme/jobs/123", "greenhouse", "acme", true},
		{"lever board", "https://jobs.lever.co/acme", "lever", "acme", true},
		{"ashby board", "https://jobs.ashbyhq.com/acme/role-slug", "ashby", "acme", true},
		{"workable board", "https://apply.workable.com/acme/", "workable", "acme", true},
		{"recruitee subdomain", "https://acme.recruitee.com/o/software-engineer", "recruitee", "acme", true},
		{"personio subdomain", "https://acme.jobs.personio.de/", "personio", "acme", true},
		{"pinpoint job url", "https://acme.pinpointhq.com/en/postings/ce6c9e5c-a2d3", "pinpoint", "acme", true},
		{"pinpoint board url", "https://acme.pinpointhq.com/", "pinpoint", "acme", true},
		{"bare pinpoint host rejected", "https://pinpointhq.com", "", "", false},
		{"pinpoint www rejected", "https://www.pinpointhq.com/", "", "", false},
		{"teamtailor job url", "https://acme.teamtailor.com/jobs/123-engineer", "teamtailor", "acme", true},
		{"teamtailor board url", "https://acme.teamtailor.com/", "teamtailor", "acme", true},
		{"bare teamtailor host rejected", "https://teamtailor.com", "", "", false},
		{"teamtailor app rejected", "https://app.teamtailor.com/", "", "", false},
		{"teamtailor www rejected", "https://www.teamtailor.com/", "", "", false},
		{"teamtailor career rejected", "https://career.teamtailor.com/", "", "", false},
		{"teamtailor api rejected", "https://api.teamtailor.com/", "", "", false},
		{"hibob job url", "https://acme.careers.hibob.com/jobs/abc-123", "hibob", "acme", true},
		{"hibob board url", "https://acme.careers.hibob.com/jobs", "hibob", "acme", true},
		{"bare hibob careers host rejected", "https://careers.hibob.com/jobs", "", "", false},
		{"hibob www rejected", "https://www.careers.hibob.com/", "", "", false},
		{"smartrecruiters job url keeps case", "https://jobs.smartrecruiters.com/Wise/744000153370939-card-disputes-specialist", "smartrecruiters", "Wise", true},
		{"smartrecruiters board url", "https://careers.smartrecruiters.com/Wise", "smartrecruiters", "Wise", true},
		{"bare smartrecruiters host rejected", "https://jobs.smartrecruiters.com/", "", "", false},
		{"aggregator rejected", "https://www.linkedin.com/jobs/view/123", "", "", false},
		{"unknown html rejected", "https://workinstartups.com/job-board", "", "", false},
		{"bare recruitee host rejected", "https://recruitee.com", "", "", false},
		{"bare greenhouse host rejected", "https://boards.greenhouse.io", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSource, gotToken, gotOK := detect.ResolveBoard(tt.url)
			if gotSource != tt.wantSource || gotToken != tt.wantToken || gotOK != tt.wantOK {
				t.Errorf("ResolveBoard(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.url, gotSource, gotToken, gotOK, tt.wantSource, tt.wantToken, tt.wantOK)
			}
		})
	}
}

func TestRewriteToATS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantURL  string
		wantType detect.ATSType
		wantOK   bool
	}{
		{
			name:     "linkedin with greenhouse externalUrl",
			input:    "https://www.linkedin.com/jobs/view/123?externalUrl=https%3A%2F%2Fboards.greenhouse.io%2Facme%2Fjobs%2F456",
			wantURL:  "https://boards.greenhouse.io/acme/jobs/456",
			wantType: detect.Greenhouse,
			wantOK:   true,
		},
		{
			name:     "linkedin without externalUrl",
			input:    "https://www.linkedin.com/jobs/view/1234567890",
			wantURL:  "",
			wantType: detect.UnknownHTML,
			wantOK:   false,
		},
		{
			name:     "indeed with lever url param",
			input:    "https://indeed.com/viewjob?url=https%3A%2F%2Fjobs.lever.co%2Facme%2Fabc-123",
			wantURL:  "https://jobs.lever.co/acme/abc-123",
			wantType: detect.Lever,
			wantOK:   true,
		},
		{
			name:     "indeed without url param",
			input:    "https://indeed.com/viewjob?jk=abc123",
			wantURL:  "",
			wantType: detect.UnknownHTML,
			wantOK:   false,
		},
		{
			name:     "non-aggregator URL",
			input:    "https://boards.greenhouse.io/acme/jobs/123",
			wantURL:  "",
			wantType: detect.UnknownHTML,
			wantOK:   false,
		},
		{
			name:     "empty string",
			input:    "",
			wantURL:  "",
			wantType: detect.UnknownHTML,
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotType, gotOK := detect.RewriteToATS(tt.input)
			if gotURL != tt.wantURL || gotType != tt.wantType || gotOK != tt.wantOK {
				t.Errorf("RewriteToATS(%q) = (%q, %v, %v), want (%q, %v, %v)",
					tt.input, gotURL, gotType, gotOK, tt.wantURL, tt.wantType, tt.wantOK)
			}
		})
	}
}
