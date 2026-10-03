package wis_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wis"
)

func newScraper(tb testing.TB, search wis.Search) *wis.Scraper {
	tb.Helper()
	tb.Setenv("DECODO_PROXY_URL", "http://user:pass@localhost:7000")
	tb.Setenv("BRIGHTDATA_PROXY_URL", "http://user:pass@localhost:7001")
	return wis.New(search)
}

func TestSnapshots(t *testing.T) {
	sourcetest.RunSnapshotTests(t, newScraper(t, wis.Search{}))
}

func FuzzParse(f *testing.F) {
	sourcetest.FuzzSnapshots(f, newScraper(f, wis.Search{}))
}

func TestSearchURL(t *testing.T) {
	tests := []struct {
		name   string
		search wis.Search
		want   string
	}{
		{
			name:   "keywords only still sorts newest first",
			search: wis.Search{Keywords: "go"},
			want:   "https://workinstartups.com/search?per_page=50&q=go&sb=date&sd=down",
		},
		{
			name: "every filter",
			search: wis.Search{Keywords: "go", Filters: map[string]string{
				"loc": "86384", "remote_only": "1", "category": "2",
				"contract": "permanent", "hours": "full_time", "salary_from": "30000",
			}},
			want: "https://workinstartups.com/search?cat=2&cti=full_time&cty=permanent&loc=86384&per_page=50&q=go&remote_only=1&sb=date&sd=down&sf=30000",
		},
		{
			name:   "invalid value is dropped",
			search: wis.Search{Keywords: "go", Filters: map[string]string{"loc": "nowhere", "hours": "full_time"}},
			want:   "https://workinstartups.com/search?cti=full_time&per_page=50&q=go&sb=date&sd=down",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wis.StartURL(tt.search); got != tt.want {
				t.Errorf("StartURL(%+v) = %q, want %q", tt.search, got, tt.want)
			}
		})
	}
}

func TestRecency(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *time.Time {
		at := now.Add(-d)
		return &at
	}
	tests := []struct {
		name string
		last *time.Time
		want string
	}{
		{"first run", nil, ""},
		{"two hours ago", ago(2 * time.Hour), "1"},
		{"thirty hours ago", ago(30 * time.Hour), "3"},
		{"four days ago", ago(4 * 24 * time.Hour), "7"},
		{"nine days ago", ago(9 * 24 * time.Hour), ""},
		{"margin pushes a boundary up a tier", ago(23*time.Hour + 30*time.Minute), "3"},
		{"future last run still keeps the margin", ago(-time.Hour), "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wis.Recency(tt.last, now); got != tt.want {
				t.Errorf("Recency = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchPage_RecencyParam(t *testing.T) {
	tests := []struct {
		name    string
		recency string
		want    string
	}{
		{"sent when set", "3", "3"},
		{"omitted when empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := newScraper(t, wis.Search{Keywords: "go", Recency: tt.recency})
			recorder := sourcetest.Respond(`<span data-cy-count="0"></span>`)
			src.Client().Transport = recorder
			if _, _, err := src.FetchPage(t.Context(), ""); err != nil {
				t.Fatal(err)
			}
			q := recorder.Last.URL.Query()
			if got := q.Get("f"); got != tt.want {
				t.Errorf("f = %q, want %q", got, tt.want)
			}
			if q.Has("f") != (tt.want != "") {
				t.Errorf("f present = %v, want %v", q.Has("f"), tt.want != "")
			}
		})
	}
}

type countingTransport struct {
	status     int
	retryAfter string
	requests   []*http.Request
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, req)
	h := http.Header{}
	if c.retryAfter != "" {
		h.Set("Retry-After", c.retryAfter)
	}
	return &http.Response{
		StatusCode: c.status,
		Status:     fmt.Sprintf("%d %s", c.status, http.StatusText(c.status)),
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(`<span data-cy-count="0"></span>`)),
	}, nil
}

func TestRequests_BrowserHeaders(t *testing.T) {
	wis.ResetLimiter()
	src := newScraper(t, wis.Search{Keywords: "go"})
	tr := &countingTransport{status: http.StatusOK}
	src.Client().Transport = tr
	if _, _, err := src.FetchPage(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := src.GetDetails(t.Context(), "https://workinstartups.com/details/1"); err == nil {
		t.Fatal("GetDetails on a listing body = nil error, want a parse error")
	}
	list, detail := tr.requests[0].Header, tr.requests[1].Header
	if !strings.Contains(list.Get("User-Agent"), "Chrome/153") || list.Get("Sec-Ch-Ua-Platform") == "" {
		t.Errorf("listing headers = %v, want Chrome UA and client hints", list)
	}
	if list.Get("Sec-Fetch-Site") != "none" || list.Get("Referer") != "" {
		t.Errorf("first request Sec-Fetch-Site/Referer = %q/%q, want none and no referer", list.Get("Sec-Fetch-Site"), list.Get("Referer"))
	}
	if detail.Get("Sec-Fetch-Site") != "same-origin" || detail.Get("Referer") != tr.requests[0].URL.String() {
		t.Errorf("detail Sec-Fetch-Site/Referer = %q/%q, want same-origin and the listing URL", detail.Get("Sec-Fetch-Site"), detail.Get("Referer"))
	}
}

func TestRateLimit(t *testing.T) {
	for _, tt := range []struct{ name, retryAfter string }{
		{"429 with Retry-After blocks later requests", "120"},
		{"429 without Retry-After blocks later requests", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wis.ResetLimiter()
			src := newScraper(t, wis.Search{Keywords: "go"})
			tr := &countingTransport{status: http.StatusTooManyRequests, retryAfter: tt.retryAfter}
			src.Client().Transport = tr

			_, _, err := src.FetchPage(t.Context(), "")
			var se *sources.StatusError
			if !errors.As(err, &se) || se.Code != http.StatusTooManyRequests {
				t.Fatalf("FetchPage = %v, want a 429 StatusError", err)
			}
			if _, _, err := src.FetchPage(t.Context(), ""); !errors.Is(err, sources.ErrDeferred) {
				t.Errorf("listing after 429 = %v, want ErrDeferred", err)
			}
			if _, err := src.GetDetails(t.Context(), "https://workinstartups.com/details/1"); !errors.Is(err, sources.ErrDeferred) {
				t.Errorf("detail after 429 = %v, want ErrDeferred", err)
			}
			if len(tr.requests) != 1 {
				t.Errorf("sent %d requests, want only the one that was refused", len(tr.requests))
			}
		})
	}
}
