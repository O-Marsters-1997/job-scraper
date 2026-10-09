package sources_test

import (
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestBoardSource_FetchPage(t *testing.T) {
	baseURL := "https://8.8.8.8"

	src := sources.NewBoardSource(sources.BoardSpec{
		Name:        "stub",
		CompanySlug: "acme",
		URL:         baseURL + "/acme",
		Parse: func([]byte) ([]dto.Job, error) {
			return []dto.Job{{Title: "acme", URL: "https://example.com/acme"}}, nil
		},
	})
	src.Client().Transport = sourcetest.Respond("ok")

	jobs, next, err := src.FetchPage(t.Context(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != "" {
		t.Errorf("next = %q, want empty", next)
	}
	if len(jobs) != 1 || jobs[0].Title != "acme" || jobs[0].Source != "stub" || jobs[0].CompanySlug != "acme" {
		t.Errorf("jobs = %v, want one stub job for acme", jobs)
	}
}

func TestBoardSource_FetchPageRejectsNonEmptyCursor(t *testing.T) {
	src := sources.NewBoardSource(sources.BoardSpec{
		Name: "stub",
		URL:  "https://8.8.8.8/acme",
		Parse: func([]byte) ([]dto.Job, error) {
			return []dto.Job{{Title: "acme"}}, nil
		},
	})
	if _, _, err := src.FetchPage(t.Context(), "again"); err == nil {
		t.Fatal("expected an error for a non-empty cursor")
	}
}

func TestBoardParserIsIndependentOfFetchMode(t *testing.T) {
	t.Setenv("DECODO_PROXY_URL", "http://user:pass@gate.decodo.com:7000")
	for _, route := range []sources.Route{sources.RouteDirect, sources.RouteResidential} {
		src := sources.NewBoardSource(sources.BoardSpec{
			Name: "stub", Route: route,
			URL: "https://8.8.8.8/jobs",
			Parse: func(body []byte) ([]dto.Job, error) {
				return []dto.Job{{Title: string(body)}}, nil
			},
		})
		src.Client().Transport = sourcetest.Respond("Engineer")
		jobs, _, err := src.FetchPage(t.Context(), "")
		if err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 1 || jobs[0].Title != "Engineer" {
			t.Fatalf("route=%v jobs=%v", route, jobs)
		}
	}
}

func TestFetchPage(t *testing.T) {
	src := sources.NewBoardSource(sources.BoardSpec{
		Name:  "stub",
		URL:   "https://8.8.8.8/acme",
		Parse: func([]byte) ([]dto.Job, error) { return nil, nil },
	})
	src.Client().Transport = identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: http.NoBody}, nil
	})
	if _, _, err := src.FetchPage(t.Context(), ""); err == nil {
		t.Fatal("FetchPage() = nil, want error for 404")
	}
}
