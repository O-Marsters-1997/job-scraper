package detect_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/detect"
)

func TestResolvePosting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		url                      string
		source, token, postingID string
	}{
		{"greenhouse boards", "https://boards.greenhouse.io/acme/jobs/1001", "greenhouse", "acme", "1001"},
		{"greenhouse job-boards", "https://job-boards.greenhouse.io/acme/jobs/1001?gh_src=x", "greenhouse", "acme", "1001"},
		{"greenhouse leading zeros", "https://boards.greenhouse.io/acme/jobs/0001001", "greenhouse", "acme", "1001"},
		{"greenhouse embed", "https://boards.greenhouse.io/embed/job_app?for=acme&token=1001", "greenhouse", "acme", "1001"},
		{"greenhouse gh_jid on board", "https://boards.greenhouse.io/acme?gh_jid=1001", "greenhouse", "acme", "1001"},
		{"lever", "https://jobs.lever.co/acme/abc123", "lever", "acme", "abc123"},
		{"lever apply", "https://jobs.lever.co/acme/abc123/apply", "lever", "acme", "abc123"},
		{"ashby", "https://jobs.ashbyhq.com/acme/2a9fc9fb-f50b-43d3-8d3b-b86bee43e510", "ashby", "acme", "2a9fc9fb-f50b-43d3-8d3b-b86bee43e510"},
		{"ashby application", "https://jobs.ashbyhq.com/acme/2a9fc9fb/application", "ashby", "acme", "2a9fc9fb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source, token, id, ok := detect.ResolvePosting(tt.url)
			if !ok || source != tt.source || token != tt.token || id != tt.postingID {
				t.Errorf("ResolvePosting(%q) = %q, %q, %q, %v; want %q, %q, %q, true",
					tt.url, source, token, id, ok, tt.source, tt.token, tt.postingID)
			}
		})
	}
}

func TestResolvePosting_Rejected(t *testing.T) {
	t.Parallel()
	for _, u := range []string{
		"https://apply.workable.com/acme/j/ABC123/",
		"https://acme.recruitee.com/o/engineer",
		"https://jobs.smartrecruiters.com/Wise/744000153370939-card",
		"https://boards.greenhouse.io/acme",
		"https://boards.greenhouse.io/acme/jobs/notanumber",
		"https://boards.greenhouse.io/embed/job_app?for=acme",
		"https://acme.com/careers?gh_jid=1001",
		"https://jobs.lever.co/acme",
		"https://jobs.ashbyhq.com/",
		"https://evilgreenhouse.io/acme/jobs/1",
		"https://example.com/acme/jobs/1",
		"://bad",
	} {
		t.Run(u, func(t *testing.T) {
			t.Parallel()
			if source, token, id, ok := detect.ResolvePosting(u); ok {
				t.Errorf("ResolvePosting(%q) = %q, %q, %q, true; want ok=false", u, source, token, id)
			}
		})
	}
}

func TestResolvePosting_MatchesAdapterSnapshots(t *testing.T) {
	t.Parallel()
	goldens, err := filepath.Glob("../worker/sources/*/snapshots/*.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	checked := map[string]int{}
	for _, path := range goldens {
		source := filepath.Base(filepath.Dir(filepath.Dir(path)))
		if source != "greenhouse" && source != "lever" && source != "ashby" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var jobs []struct{ URL, ProviderPostingID string }
		if err := json.Unmarshal(raw, &jobs); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		for _, j := range jobs {
			checked[source]++
			gotSource, _, id, ok := detect.ResolvePosting(j.URL)
			if !ok || gotSource != source || id != j.ProviderPostingID {
				t.Errorf("%s: ResolvePosting(%q) = %q, %q, %v; want %q, %q, true",
					filepath.Base(path), j.URL, gotSource, id, ok, source, j.ProviderPostingID)
			}
		}
	}
	for _, s := range []string{"greenhouse", "lever", "ashby"} {
		if checked[s] == 0 {
			t.Errorf("no %s snapshot postings checked", s)
		}
	}
}
