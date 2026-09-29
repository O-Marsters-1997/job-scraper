package jobsearch_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
)

func TestListSources(t *testing.T) {
	got, err := jobsearch.NewService(nil, nil).ListSources(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("want at least one registered source")
	}
}

func TestResolveBoard(t *testing.T) {
	svc := jobsearch.NewService(nil, nil)

	_, err := svc.ResolveBoard(context.Background(), "user-1", dto.ResolveBoardQuery{})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("empty url: status = %v, ok = %v", status, ok)
	}

	_, err = svc.ResolveBoard(context.Background(), "user-1", dto.ResolveBoardQuery{URL: "https://example.com/careers"})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnprocessable.Status() {
		t.Fatalf("unresolvable url: status = %v, ok = %v", status, ok)
	}

	got, err := svc.ResolveBoard(context.Background(), "user-1", dto.ResolveBoardQuery{URL: "https://boards.greenhouse.io/acme"})
	if err != nil {
		t.Fatal(err)
	}
	want := dto.ResolvedURL{
		Kind: "ats", Source: "greenhouse", Value: "acme",
		Filters: map[string]string{}, Dropped: []string{}, URL: "https://boards.greenhouse.io/acme",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("ResolveBoard(ats) mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveBoard_Search(t *testing.T) {
	svc := jobsearch.NewService(nil, nil)
	got, err := svc.ResolveBoard(context.Background(), "user-1", dto.ResolveBoardQuery{
		URL: "https://www.linkedin.com/jobs/search-results/?keywords=go&f_WT=2&currentJobId=1&start=25",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := dto.ResolvedURL{
		Kind: "search", Source: "linkedin", Value: "go",
		Filters: map[string]string{"arrangement": "2"}, Dropped: []string{"currentJobId"},
		URL: "https://www.linkedin.com/jobs/search/?f_WT=2&keywords=go",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("ResolveBoard(search) mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveBoard_UnsupportedAndUnrecognisedDiffer(t *testing.T) {
	svc := jobsearch.NewService(nil, nil)
	messages := map[string]string{}
	for _, url := range []string{"https://remoteok.com/remote-go-jobs", "https://example.com/careers"} {
		_, err := svc.ResolveBoard(context.Background(), "user-1", dto.ResolveBoardQuery{URL: url})
		if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindUnprocessable.Status() {
			t.Fatalf("%s: status = %v, ok = %v", url, status, ok)
		}
		messages[url] = err.Error()
	}
	if messages["https://remoteok.com/remote-go-jobs"] == messages["https://example.com/careers"] {
		t.Errorf("unsupported and unrecognised share a message: %q", messages)
	}
}
