package sources_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type boardRoundTrip func(*http.Request) (*http.Response, error)

func (f boardRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestBoardSource_Iterate_ContinuesPastFailingToken(t *testing.T) {
	baseURL := "https://8.8.8.8"

	var fetched []string
	src := sources.NewBoardSource([]string{"good1", "bad", "good2"}, sources.BoardSpec{
		Name:      "stub",
		URLPrefix: baseURL,
		URL:       func(token string) string { return baseURL + "/" + token },
		Parse: func(body []byte, token string) ([]dto.Job, error) {
			return []dto.Job{{Title: token, URL: "https://example.com/" + token}}, nil
		},
	})
	src.Client().Transport = boardRoundTrip(func(r *http.Request) (*http.Response, error) {
		status := http.StatusOK
		if r.URL.Path == "/bad" {
			status = http.StatusInternalServerError
		}
		return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})

	var done []string
	var doneURLs [][]string
	src.WithDone(func(_ context.Context, token string, urls []string) {
		done = append(done, token)
		doneURLs = append(doneURLs, urls)
	})

	err := src.Iterate(context.Background(), func(_ context.Context, jobs []dto.Job) (bool, error) {
		for _, j := range jobs {
			fetched = append(fetched, j.Title)
		}
		return false, nil
	})

	if err == nil {
		t.Fatal("expected a joined error for the failing token, got nil")
	}
	if want := []string{"good1", "good2"}; !equalSlices(fetched, want) {
		t.Errorf("fetched = %v, want %v", fetched, want)
	}
	if want := []string{"good1", "good2"}; !equalSlices(done, want) {
		t.Errorf("done called with %v, want %v (bad token should not call done)", done, want)
	}
	if len(doneURLs) != 2 || doneURLs[0][0] != "https://example.com/good1" || doneURLs[1][0] != "https://example.com/good2" {
		t.Errorf("done urls = %v, want per-token job URLs", doneURLs)
	}
}

func TestBoardSource_Iterate_AllSucceedNoError(t *testing.T) {
	baseURL := "https://8.8.8.8"

	src := sources.NewBoardSource([]string{"a", "b"}, sources.BoardSpec{
		Name:      "stub",
		URLPrefix: baseURL,
		URL:       func(token string) string { return baseURL + "/" + token },
		Parse: func(_ []byte, token string) ([]dto.Job, error) {
			return []dto.Job{{Title: token, URL: "https://example.com/" + token}}, nil
		},
	})
	src.Client().Transport = boardRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})

	err := src.Iterate(context.Background(), func(_ context.Context, _ []dto.Job) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBoardParserIsIndependentOfFetchMode(t *testing.T) {
	t.Setenv("BRIGHTDATA_PROXY_URL", "http://user:pass@brd.superproxy.io:33335")
	for _, useProxy := range []bool{false, true} {
		src := sources.NewBoardSource([]string{"board"}, sources.BoardSpec{
			Name: "stub", UseProxy: useProxy,
			URL: func(string) string { return "https://8.8.8.8/jobs" },
			Parse: func(body []byte, _ string) ([]dto.Job, error) {
				return []dto.Job{{Title: string(body)}}, nil
			},
		})
		src.Client().Transport = boardRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("Engineer"))}, nil
		})
		err := src.Iterate(context.Background(), func(_ context.Context, jobs []dto.Job) (bool, error) {
			if len(jobs) != 1 || jobs[0].Title != "Engineer" {
				t.Fatalf("useProxy=%t jobs=%v", useProxy, jobs)
			}
			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
