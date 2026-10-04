package workablesearch

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

func TestHarvester_Harvest(t *testing.T) {
	pages := map[string][]byte{}
	for token, file := range map[string]string{"": "page1.json", "page2token==": "page2.json"} {
		body, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		pages[token] = body
	}
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.UserAgent()
		body, ok := pages[r.URL.Query().Get("pageToken")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	got, err := New(srv.Client(), srv.URL+"/api/v1/jobs?location=United%20Kingdom").Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := discover.Harvest{
		Boards: []discover.Board{{Source: "workable", Token: "acme-direct"}},
		Companies: []discover.Company{
			{Slug: "bask-health", Name: "Bask Health", Board: discover.Board{Source: "workable", Token: "bask-health"}},
			{Slug: "orchard", Name: "Orchard", Board: discover.Board{Source: "workable", Token: "orchard"}},
			{Slug: "pact-coffee", Name: "Pact Coffee", Board: discover.Board{Source: "workable", Token: "pact-coffee"}},
		},
		Skipped: 1,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Harvest (-want +got):\n%s", diff)
	}
	if gotUA != userAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, userAgent)
	}
}
