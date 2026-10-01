package wttj

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestParseSitemap(t *testing.T) {
	f, err := os.Open("snapshots/sitemap.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	jobIDs, companyNames, err := parseSitemap(f)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]string{"QKPz2OAK", "kamFi6WS", "4Mxoqdvu", "0iAUvrEd"}, jobIDs); diff != "" {
		t.Errorf("job IDs (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"Ello", "Koiji", "iwoca"}, companyNames); diff != "" {
		t.Errorf("company names (-want +got):\n%s", diff)
	}
}

type recordingSyncer struct{ called bool }

func (s *recordingSyncer) SyncWTTJSitemap(context.Context, []string, []string) (dto.SitemapDiff, error) {
	s.called = true
	return dto.SitemapDiff{}, nil
}

func TestTrackerRunOnce(t *testing.T) {
	sitemap, err := os.ReadFile("snapshots/sitemap.xml")
	if err != nil {
		t.Fatal(err)
	}
	var status int
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.UserAgent()
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = w.Write(sitemap)
		}
	}))
	defer srv.Close()

	t.Run("429 returns ErrBlocked without syncing", func(t *testing.T) {
		status = http.StatusTooManyRequests
		syncer := &recordingSyncer{}
		tr := NewTracker(syncer)
		tr.url = srv.URL
		if err := tr.RunOnce(context.Background()); !errors.Is(err, ErrBlocked) {
			t.Errorf("RunOnce() = %v, want ErrBlocked", err)
		}
		if syncer.called {
			t.Error("RunOnce synced after a 429")
		}
	})

	t.Run("200 syncs with the honest user agent", func(t *testing.T) {
		status = http.StatusOK
		syncer := &recordingSyncer{}
		tr := NewTracker(syncer)
		tr.url = srv.URL
		if err := tr.RunOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		if !syncer.called || gotUA != userAgent {
			t.Errorf("synced = %v, user agent = %q, want synced with %q", syncer.called, gotUA, userAgent)
		}
	})
}
