package sources_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type boardRoundTrip func(*http.Request) (*http.Response, error)

func (f boardRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestBoardSource_FetchPage(t *testing.T) {
	baseURL := "https://8.8.8.8"

	src := sources.NewBoardSource("acme", sources.BoardSpec{
		Name: "stub",
		URL:  func(token string) string { return baseURL + "/" + token },
		Parse: func(body []byte, token string) ([]dto.Job, error) {
			return []dto.Job{{Title: token, URL: "https://example.com/" + token}}, nil
		},
	})
	src.Client().Transport = boardRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})

	jobs, next, err := src.FetchPage(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != "" {
		t.Errorf("next = %q, want empty", next)
	}
	if len(jobs) != 1 || jobs[0].Title != "acme" {
		t.Errorf("jobs = %v, want one job for acme", jobs)
	}
}

func TestBoardSource_FetchPageRejectsNonEmptyCursor(t *testing.T) {
	src := sources.NewBoardSource("acme", sources.BoardSpec{
		Name: "stub",
		URL:  func(token string) string { return "https://8.8.8.8/" + token },
		Parse: func(body []byte, token string) ([]dto.Job, error) {
			return []dto.Job{{Title: token}}, nil
		},
	})
	if _, _, err := src.FetchPage(context.Background(), "again"); err == nil {
		t.Fatal("expected an error for a non-empty cursor")
	}
}

func TestBoardParserIsIndependentOfFetchMode(t *testing.T) {
	t.Setenv("BRIGHTDATA_PROXY_URL", "http://user:pass@brd.superproxy.io:33335")
	for _, useProxy := range []bool{false, true} {
		src := sources.NewBoardSource("board", sources.BoardSpec{
			Name: "stub", UseProxy: useProxy,
			URL: func(string) string { return "https://8.8.8.8/jobs" },
			Parse: func(body []byte, _ string) ([]dto.Job, error) {
				return []dto.Job{{Title: string(body)}}, nil
			},
		})
		src.Client().Transport = boardRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("Engineer"))}, nil
		})
		jobs, _, err := src.FetchPage(context.Background(), "")
		if err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 1 || jobs[0].Title != "Engineer" {
			t.Fatalf("useProxy=%t jobs=%v", useProxy, jobs)
		}
	}
}
