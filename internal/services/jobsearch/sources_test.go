package jobsearch_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
)

func TestListSources(t *testing.T) {
	got, err := jobsearch.NewService(nil, nil, nil).ListSources(t.Context(), userID)
	if err != nil {
		t.Fatalf("ListSources() err = %v", err)
	}
	if len(got) == 0 {
		t.Fatal("ListSources() = none, want at least one registered source")
	}
}

func TestResolveBoard(t *testing.T) {
	svc := jobsearch.NewService(nil, nil, nil)

	t.Run("resolves", func(t *testing.T) {
		tests := []struct {
			name string
			url  string
			want dto.ResolvedURL
		}{
			{
				name: "ats board",
				url:  acmeBoard,
				want: dto.ResolvedURL{
					Kind: "ats", Source: "greenhouse", Value: "acme",
					Filters: map[string]string{}, Dropped: []string{}, URL: acmeBoard,
				},
			},
			{
				name: "search",
				url:  "https://www.linkedin.com/jobs/search-results/?keywords=go&f_WT=2&currentJobId=1&start=25",
				want: dto.ResolvedURL{
					Kind: "search", Source: "linkedin", Value: "go",
					Filters: map[string]string{"arrangement": "2"}, Dropped: []string{"currentJobId"},
					URL: "https://www.linkedin.com/jobs/search/?f_WT=2&keywords=go",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := svc.ResolveBoard(t.Context(), userID, dto.ResolveBoardQuery{URL: tt.url})
				if err != nil {
					t.Fatalf("ResolveBoard(%q) err = %v", tt.url, err)
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("ResolveBoard(%q) (-want +got):\n%s", tt.url, diff)
				}
			})
		}
	})

	t.Run("rejects", func(t *testing.T) {
		tests := []struct {
			name     string
			url      string
			wantKind apperr.Kind
		}{
			{name: "empty url", wantKind: apperr.KindInvalid},
			{name: "unrecognised url", url: "https://example.com/careers", wantKind: apperr.KindUnprocessable},
			{name: "unsupported source", url: "https://remoteok.com/remote-go-jobs", wantKind: apperr.KindUnprocessable},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := svc.ResolveBoard(t.Context(), userID, dto.ResolveBoardQuery{URL: tt.url})
				if !apperr.IsKind(err, tt.wantKind) {
					t.Errorf("ResolveBoard(%q) err = %v, want kind %v", tt.url, err, tt.wantKind)
				}
			})
		}
	})
}
