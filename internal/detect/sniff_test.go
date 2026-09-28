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
