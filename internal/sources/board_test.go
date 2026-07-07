package sources_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

func TestBoardSource_Iterate_ContinuesPastFailingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/good1", "/good2":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/bad":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	var fetched []string
	src := sources.NewBoardSource([]string{"good1", "bad", "good2"}, sources.BoardSpec{
		Name:      "stub",
		URLPrefix: srv.URL,
		URL:       func(token string) string { return srv.URL + "/" + token },
		Parse: func(body []byte, token string) ([]dto.Job, error) {
			return []dto.Job{{Title: token, URL: "https://example.com/" + token}}, nil
		},
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	src := sources.NewBoardSource([]string{"a", "b"}, sources.BoardSpec{
		Name:      "stub",
		URLPrefix: srv.URL,
		URL:       func(token string) string { return srv.URL + "/" + token },
		Parse: func(_ []byte, token string) ([]dto.Job, error) {
			return []dto.Job{{Title: token, URL: "https://example.com/" + token}}, nil
		},
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
