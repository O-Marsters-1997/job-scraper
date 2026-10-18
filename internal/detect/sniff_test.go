package detect_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/detect"
)

func TestSniffATS(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantSource string
		wantToken  string
		wantOK     bool
	}{
		{
			name:       "greenhouse embed job_board for= param",
			body:       `<script>window.config = {board: "https://boards.greenhouse.io/embed/job_board?for=acmecorp&b=true"};</script>`,
			wantSource: "greenhouse",
			wantToken:  "acmecorp",
			wantOK:     true,
		},
		{
			name:       "lever board embedded in inline script",
			body:       `<script>var apiUrl = "https://jobs.lever.co/acme/postings.json";</script>`,
			wantSource: "lever",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "ashby board embedded in SPA config blob",
			body:       `<div data-config='{"boardUrl":"https://jobs.ashbyhq.com/acme"}'></div>`,
			wantSource: "ashby",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "greenhouse embed js loader",
			body:       `<script src="https://boards.greenhouse.io/embed/job_board/js?for=acme"></script>`,
			wantSource: "greenhouse",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "ashby posting api",
			body:       `fetch("https://api.ashbyhq.com/posting-api/job-board/acme")`,
			wantSource: "ashby",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "workable",
			body:       `<a href="https://apply.workable.com/acme/">`,
			wantSource: "workable",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "recruitee skips reserved subdomain",
			body:       `<img src="https://www.recruitee.com/logo.png"><script src="https://acme.recruitee.com/widget.js">`,
			wantSource: "recruitee",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "personio",
			body:       `<script src="https://acme.jobs.personio.de/js/widget.js">`,
			wantSource: "personio",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "pinpoint",
			body:       `<script src="https://acme.pinpointhq.com/widget.js">`,
			wantSource: "pinpoint",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "teamtailor",
			body:       `<iframe src="https://acme.teamtailor.com/jobs">`,
			wantSource: "teamtailor",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "hibob",
			body:       `<iframe src="https://acme.careers.hibob.com/">`,
			wantSource: "hibob",
			wantToken:  "acme",
			wantOK:     true,
		},
		{
			name:       "smartrecruiters widget",
			body:       `<script src="https://widget.smartrecruiters.com/Acme/widget.js">`,
			wantSource: "smartrecruiters",
			wantToken:  "Acme",
			wantOK:     true,
		},
		{
			name:       "workday detected but not recorded",
			body:       `<script>var board = "https://acme.myworkdayjobs.com/en-US/careers";</script>`,
			wantSource: "",
			wantToken:  "",
			wantOK:     false,
		},
		{
			name:       "no markers present",
			body:       `<html><body><h1>Careers at Acme</h1></body></html>`,
			wantSource: "",
			wantToken:  "",
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSource, gotToken, gotOK := detect.SniffATS([]byte(tt.body))
			if gotSource != tt.wantSource || gotToken != tt.wantToken || gotOK != tt.wantOK {
				t.Errorf("SniffATS(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.body, gotSource, gotToken, gotOK, tt.wantSource, tt.wantToken, tt.wantOK)
			}
		})
	}
}

func FuzzSniffATS(f *testing.F) {
	for _, seed := range []string{
		`<script>window.config = {board: "https://boards.greenhouse.io/embed/job_board?for=acmecorp&b=true"};</script>`,
		`<script>var apiUrl = "https://jobs.lever.co/acme/postings.json";</script>`,
		`<div data-config='{"boardUrl":"https://jobs.ashbyhq.com/acme"}'></div>`,
		`<script>var board = "https://acme.myworkdayjobs.com/en-US/careers";</script>`,
		`<html><body><h1>Careers at Acme</h1></body></html>`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(_ *testing.T, body []byte) {
		detect.SniffATS(body)
	})
}
