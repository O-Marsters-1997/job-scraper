package hiringcafe_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/hiringcafe"
)

type decodoRequest struct {
	URL       string `json:"url"`
	Geo       string `json:"geo"`
	Headless  string `json:"headless"`
	SessionID string `json:"session_id"`
	Pool      string `json:"proxy_pool"`
}

type decodoServer struct {
	mu       sync.Mutex
	requests []decodoRequest
	auth     string
	respond  func(decodoRequest) (status int, content string)
}

func newDecodo(t *testing.T, respond func(decodoRequest) (int, string)) (*decodoServer, *hiringcafe.API) {
	t.Helper()
	d := &decodoServer{respond: respond}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req decodoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		d.mu.Lock()
		d.requests = append(d.requests, req)
		d.auth = r.Header.Get("Authorization")
		d.mu.Unlock()
		status, content := respond(req)
		w.WriteHeader(status)
		if status != http.StatusOK {
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"content": content, "status_code": 200}}}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return d, hiringcafe.NewAPI(srv.URL, "tok", sources.NewGate(0, time.Minute))
}

const searchStateJSON = `{"locations":[{"id":"x"}]}`

func nextDataPage(buildID, props string) string {
	return fmt.Sprintf(`<html><script id="__NEXT_DATA__" type="application/json">{"buildId":%q,"props":{"pageProps":%s}}</script></html>`, buildID, props)
}

func TestAPI_Search(t *testing.T) {
	t.Parallel()
	state := json.RawMessage(searchStateJSON)

	t.Run("page 0 renders the classic page and returns its pageProps", func(t *testing.T) {
		t.Parallel()
		d, api := newDecodo(t, func(decodoRequest) (int, string) {
			return 200, nextDataPage("b1", `{"ssrHits":[]}`)
		})
		got, err := api.Search(t.Context(), state, 0)
		if err != nil {
			t.Fatalf("Search() err = %v", err)
		}
		if diff := cmp.Diff(`{"ssrHits":[]}`, string(got)); diff != "" {
			t.Errorf("Search() (-want +got):\n%s", diff)
		}
		req := d.requests[0]
		if req.Headless != "html" || req.Geo != "GB" || req.Pool != "premium" || d.auth != "Bearer tok" {
			t.Errorf("request = %+v auth %q, want a rendered premium GB request with the bearer token", req, d.auth)
		}
		if !strings.HasPrefix(req.URL, "https://hiringcafe.com/classic?searchState=") {
			t.Errorf("url = %q, want the classic page", req.URL)
		}
	})

	t.Run("page N uses the build id from page 0 without rendering", func(t *testing.T) {
		t.Parallel()
		d, api := newDecodo(t, func(r decodoRequest) (int, string) {
			if strings.Contains(r.URL, "/_next/data/") {
				return 200, `{"pageProps":{"ssrIsLastPage":true}}`
			}
			return 200, nextDataPage("b7", `{}`)
		})
		if _, err := api.Search(t.Context(), state, 0); err != nil {
			t.Fatalf("Search(0) err = %v", err)
		}
		got, err := api.Search(t.Context(), state, 2)
		if err != nil {
			t.Fatalf("Search(2) err = %v", err)
		}
		if diff := cmp.Diff(`{"ssrIsLastPage":true}`, string(got)); diff != "" {
			t.Errorf("Search(2) (-want +got):\n%s", diff)
		}
		req := d.requests[1]
		if !strings.Contains(req.URL, "/_next/data/b7/classic.json?searchState=") || !strings.HasSuffix(req.URL, "&page=2") || req.Headless != "" {
			t.Errorf("request = %+v, want the b7 data url for page 2 without rendering", req)
		}
	})

	t.Run("page N without a known build id loads page 0 first", func(t *testing.T) {
		t.Parallel()
		d, api := newDecodo(t, func(r decodoRequest) (int, string) {
			if strings.Contains(r.URL, "/_next/data/") {
				return 200, `{"pageProps":{}}`
			}
			return 200, nextDataPage("b9", `{}`)
		})
		if _, err := api.Search(t.Context(), state, 3); err != nil {
			t.Fatalf("Search(3) err = %v", err)
		}
		if len(d.requests) != 2 || !strings.Contains(d.requests[1].URL, "/_next/data/b9/") {
			t.Errorf("requests = %+v, want page 0 then the b9 data url", d.requests)
		}
	})

	t.Run("a refused page N is retried rendered on a fresh session", func(t *testing.T) {
		t.Parallel()
		d, api := newDecodo(t, func(r decodoRequest) (int, string) {
			switch {
			case !strings.Contains(r.URL, "/_next/data/"):
				return 200, nextDataPage("b1", `{}`)
			case r.Headless == "":
				return 500, ""
			default:
				return 200, `<html><body><pre>{"pageProps":{"ssrHits":[]}}</pre></body></html>`
			}
		})
		if _, err := api.Search(t.Context(), state, 0); err != nil {
			t.Fatalf("Search(0) err = %v", err)
		}
		got, err := api.Search(t.Context(), state, 1)
		if err != nil {
			t.Fatalf("Search(1) err = %v", err)
		}
		if diff := cmp.Diff(`{"ssrHits":[]}`, string(got)); diff != "" {
			t.Errorf("Search(1) (-want +got):\n%s", diff)
		}
		plain, rendered := d.requests[1], d.requests[2]
		if rendered.Headless != "html" || plain.SessionID == rendered.SessionID {
			t.Errorf("retry = %+v after %+v, want a rendered request on a new session", rendered, plain)
		}
	})

	t.Run("a block that outlasts the retries defers the task", func(t *testing.T) {
		t.Parallel()
		_, api := newDecodo(t, func(decodoRequest) (int, string) { return 500, "" })
		_, err := api.Search(t.Context(), state, 0)
		var deferred *sources.DeferredError
		if !errors.As(err, &deferred) {
			t.Fatalf("Search() err = %v, want a *sources.DeferredError", err)
		}
	})

	t.Run("a state without locations is rejected", func(t *testing.T) {
		t.Parallel()
		d, api := newDecodo(t, func(decodoRequest) (int, string) { return 200, "" })
		if _, err := api.Search(t.Context(), json.RawMessage(`{}`), 0); err == nil {
			t.Fatal("Search() err = nil, want an error")
		}
		if len(d.requests) != 0 {
			t.Errorf("sent %d requests, want none", len(d.requests))
		}
	})
}

func TestAPI_Description(t *testing.T) {
	t.Parallel()

	t.Run("fetches on a fresh session without rendering", func(t *testing.T) {
		t.Parallel()
		d, api := newDecodo(t, func(decodoRequest) (int, string) { return 200, `{"job":{}}` })
		for range 2 {
			got, err := api.Description(t.Context(), "a b")
			if err != nil || string(got) != `{"job":{}}` {
				t.Fatalf("Description() = %q, %v, want the json body", got, err)
			}
		}
		first, second := d.requests[0], d.requests[1]
		if first.URL != "https://hiringcafe.com/api/job-description?id=a+b" || first.Headless != "" {
			t.Errorf("request = %+v, want the escaped plain description url", first)
		}
		if first.SessionID == "" || first.SessionID == second.SessionID {
			t.Errorf("sessions %q and %q, want distinct non-empty ids", first.SessionID, second.SessionID)
		}
	})

	t.Run("a failed session is retried on a new one", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		d, api := newDecodo(t, func(decodoRequest) (int, string) {
			if calls.Add(1) == 1 {
				return 400, ""
			}
			return 200, `{"job":{}}`
		})
		if _, err := api.Description(t.Context(), "x"); err != nil {
			t.Fatalf("Description() err = %v", err)
		}
		if d.requests[0].SessionID == d.requests[1].SessionID {
			t.Error("retry reused the failed session")
		}
	})

	t.Run("a removed job is gone, not a block", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if _, err := w.Write([]byte(`{"results":[{"content":"","status_code":404}]}`)); err != nil {
				t.Errorf("write: %v", err)
			}
		}))
		t.Cleanup(srv.Close)
		api := hiringcafe.NewAPI(srv.URL, "tok", sources.NewGate(0, time.Minute))
		_, err := api.Description(t.Context(), "x")
		if !errors.Is(err, sources.ErrGone) || errors.Is(err, sources.ErrDeferred) {
			t.Errorf("Description() err = %v, want ErrGone without a deferral", err)
		}
	})

	t.Run("a challenge body is refused", func(t *testing.T) {
		t.Parallel()
		_, api := newDecodo(t, func(decodoRequest) (int, string) { return 200, `{"code":"cloudflare_challenge"}` })
		if _, err := api.Description(t.Context(), "x"); !errors.Is(err, sources.ErrDeferred) {
			t.Errorf("Description() err = %v, want a deferral", err)
		}
	})
}
