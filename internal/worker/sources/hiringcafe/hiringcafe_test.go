package hiringcafe_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/hiringcafe"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

type fakeTab struct {
	body     []byte
	err      error
	state    json.RawMessage
	page     int
	descID   string
	descBody []byte
}

func (f *fakeTab) Search(_ context.Context, state json.RawMessage, page int) ([]byte, error) {
	f.state, f.page = state, page
	return f.body, f.err
}

func (f *fakeTab) Description(_ context.Context, id string) ([]byte, error) {
	f.descID = id
	return f.descBody, f.err
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("snapshots/" + name)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

func TestScraper_FetchPage(t *testing.T) {
	t.Parallel()
	state := json.RawMessage(`{"locations":[{"id":"x"}],"sortBy":"date"}`)

	t.Run("maps a captured page to cards", func(t *testing.T) {
		t.Parallel()
		tab := &fakeTab{body: readFixture(t, "page.json")}
		jobs, next, err := hiringcafe.New(state, tab).FetchPage(t.Context(), "")
		if err != nil {
			t.Fatalf("FetchPage() err = %v", err)
		}
		sourcetest.MatchGolden(t, "page.json", jobs)
		if next != "1" {
			t.Errorf("next = %q, want %q", next, "1")
		}
		if string(tab.state) != string(state) || tab.page != 0 {
			t.Errorf("Search(%s, %d), want the target's state and page 0", tab.state, tab.page)
		}
	})

	t.Run("the cursor is the page number", func(t *testing.T) {
		t.Parallel()
		tab := &fakeTab{body: readFixture(t, "page.json")}
		_, next, err := hiringcafe.New(state, tab).FetchPage(t.Context(), "4")
		if err != nil || tab.page != 4 || next != "5" {
			t.Errorf("FetchPage(4) = next %q, err %v with page %d; want next 5 from page 4", next, err, tab.page)
		}
	})

	t.Run("the last page has no cursor", func(t *testing.T) {
		t.Parallel()
		tab := &fakeTab{body: []byte(`{"ssrHits":[],"ssrIsLastPage":true}`)}
		jobs, next, err := hiringcafe.New(state, tab).FetchPage(t.Context(), "2")
		if err != nil || len(jobs) != 0 || next != "" {
			t.Errorf("FetchPage() = %v, %q, %v; want no jobs, no cursor, no error", jobs, next, err)
		}
	})

	t.Run("a blocked tab defers", func(t *testing.T) {
		t.Parallel()
		until := time.Now().Add(time.Minute)
		tab := &fakeTab{err: &sources.DeferredError{Until: until}}
		_, _, err := hiringcafe.New(state, tab).FetchPage(t.Context(), "")
		if _, ok := sources.BlockedUntil(err, time.Now()); !ok {
			t.Errorf("FetchPage() err = %v, want a blocked error", err)
		}
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name   string
			body   string
			cursor string
		}{
			{"ssrError", `{"ssrError":"boom"}`, ""},
			{"invalid json", `<html>`, ""},
			{"bad cursor", `{}`, "abc"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, _, err := hiringcafe.New(state, &fakeTab{body: []byte(tt.body)}).FetchPage(t.Context(), tt.cursor)
				if err == nil {
					t.Error("FetchPage() err = nil, want an error")
				}
			})
		}
	})
}

func TestScraper_GetDetails(t *testing.T) {
	t.Parallel()
	t.Run("fills the description by posting id", func(t *testing.T) {
		t.Parallel()
		tab := &fakeTab{descBody: readFixture(t, "description.json")}
		card := dto.Job{Title: "Backend Engineer", ProviderPostingID: "hc-1001"}
		got, err := hiringcafe.New(nil, tab).GetDetails(t.Context(), card)
		if err != nil {
			t.Fatalf("GetDetails() err = %v", err)
		}
		want := card
		want.Description = "<p>Build the platform.</p>"
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GetDetails() mismatch (-want +got):\n%s", diff)
		}
		if tab.descID != "hc-1001" {
			t.Errorf("Description(%q), want hc-1001", tab.descID)
		}
	})

	t.Run("a card without a posting id is an error", func(t *testing.T) {
		t.Parallel()
		if _, err := hiringcafe.New(nil, &fakeTab{}).GetDetails(t.Context(), dto.Job{}); err == nil {
			t.Error("GetDetails() err = nil, want an error")
		}
	})

	t.Run("tab errors pass through", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("boom")
		_, err := hiringcafe.New(nil, &fakeTab{err: boom}).GetDetails(t.Context(), dto.Job{ProviderPostingID: "x"})
		if !errors.Is(err, boom) {
			t.Errorf("GetDetails() err = %v, want %v", err, boom)
		}
	})
}

func TestScraper_Cfg(t *testing.T) {
	t.Parallel()
	cfg := hiringcafe.New(nil, &fakeTab{}).Cfg()
	if cfg.Name != "hiringcafe" || cfg.Route != sources.RouteResidential || !cfg.NewestFirst {
		t.Errorf("Cfg() = %+v, want a newest-first residential hiringcafe source", cfg)
	}
}

func TestRecency(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *time.Time { t := now.Add(-d); return &t }

	tests := []struct {
		name string
		in   string
		last *time.Time
		want string
	}{
		{"first run only forces date sort", `{"sortBy":"relevance","dateFetchedPastNDays":61}`, nil, `{"dateFetchedPastNDays":61,"sortBy":"date"}`},
		{"narrows to whole days plus margin", `{"dateFetchedPastNDays":61}`, ago(30 * time.Hour), `{"dateFetchedPastNDays":2,"sortBy":"date"}`},
		{"never wider than the pasted window", `{"dateFetchedPastNDays":3}`, ago(10 * 24 * time.Hour), `{"dateFetchedPastNDays":3,"sortBy":"date"}`},
		{"sets a window when none was pasted", `{"locations":[]}`, ago(time.Hour), `{"dateFetchedPastNDays":1,"locations":[],"sortBy":"date"}`},
		{"not an object is untouched", `engineer`, ago(time.Hour), `engineer`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hiringcafe.Recency(json.RawMessage(tt.in), tt.last, now)
			if string(got) != tt.want {
				t.Errorf("Recency(%s) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}
