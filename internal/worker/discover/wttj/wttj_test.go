package wttj_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/wttj"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	wttjsource "github.com/ollymarsters/job-scraper/internal/worker/sources/wttj"
)

type graphql struct {
	mu        sync.Mutex
	responses []func(w http.ResponseWriter)
	calls     int
}

func (g *graphql) RoundTrip(r *http.Request) (*http.Response, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	rec := httptest.NewRecorder()
	g.responses[min(g.calls, len(g.responses)-1)](rec)
	g.calls++
	return rec.Result(), nil
}

func body(s string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) { _, _ = w.Write([]byte(s)) }
}

func status(code int) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) { w.WriteHeader(code) }
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("snapshots", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func harvester(t *testing.T, sitemap string, api *graphql) *wttj.Harvester {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sitemap))
	}))
	t.Cleanup(srv.Close)
	client := wttjsource.NewClient(0)
	client.Client().Transport = api
	return wttj.New(srv.Client(), srv.URL, client)
}

func sitemapOf(n int) string {
	var b strings.Builder
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for i := range n {
		fmt.Fprintf(&b, "<url><loc>https://app.welcometothejungle.com/jobs/job%d</loc></url>", i)
	}
	b.WriteString("</urlset>")
	return b.String()
}

func ukEngineeringJobs(n int) string {
	data := map[string]any{}
	for i := range n {
		data[fmt.Sprintf("j%d", i)] = map[string]any{
			"actualOriginalUrl":   fmt.Sprintf("https://jobs.ashbyhq.com/acme/%d", i),
			"locationPreferences": []map[string]string{{"location": "LONDON"}},
			"function":            map[string]string{"value": "Software Engineering"},
			"company":             map[string]string{"name": "Acme", "urlSafeName": "Acme"},
		}
	}
	b, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestHarvester_Harvest(t *testing.T) {
	t.Parallel()
	t.Run("UK engineering jobs yield their ATS board, or their WTTJ company without one", func(t *testing.T) {
		t.Parallel()
		h := harvester(t, fixture(t, "sitemap.xml"), &graphql{responses: []func(http.ResponseWriter){body(fixture(t, "listings.json"))}})
		got, err := h.Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		want := discover.Harvest{
			Boards: []discover.Board{{Source: "ashby", Token: "acme"}},
			Companies: []discover.Company{
				{Slug: "steno-labs", Name: "Steno Labs", Board: discover.Board{Source: "wttj", Token: "Steno-Labs"}},
			},
			Skipped: 1,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
	})

	t.Run("a block after some listings were fetched keeps them", func(t *testing.T) {
		t.Parallel()
		eligible := ukEngineeringJobs(100)
		api := &graphql{responses: []func(http.ResponseWriter){body(eligible), body(eligible), status(http.StatusTooManyRequests)}}
		got, err := harvester(t, sitemapOf(300), api).Harvest(t.Context())
		if err != nil {
			t.Fatalf("Harvest error = %v, want the fetched listings kept", err)
		}
		if diff := cmp.Diff([]discover.Board{{Source: "ashby", Token: "acme"}}, got.Boards); diff != "" {
			t.Errorf("Harvest boards (-want +got):\n%s", diff)
		}
	})

	t.Run("a block on the first batch fails the harvest", func(t *testing.T) {
		t.Parallel()
		api := &graphql{responses: []func(http.ResponseWriter){status(http.StatusTooManyRequests)}}
		if _, err := harvester(t, sitemapOf(150), api).Harvest(t.Context()); !errors.Is(err, sources.ErrDeferred) {
			t.Errorf("Harvest error = %v, want ErrDeferred", err)
		}
	})
}
