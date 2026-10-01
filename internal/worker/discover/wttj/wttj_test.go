package wttj

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

func TestParseSitemap(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("snapshots", "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}

	tokens, skipped, err := parseSitemap(body)
	if err != nil {
		t.Fatal(err)
	}

	got, err := json.MarshalIndent(struct {
		Tokens  []string
		Skipped int
	}{tokens, skipped}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("snapshots", "sitemap.golden.json")
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	var wantV, gotV any
	if err := json.Unmarshal(want, &wantV); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &gotV); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantV, gotV); diff != "" {
		t.Errorf("parseSitemap (-want +got):\n%s", diff)
	}
}

func TestHarvester_Harvest(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("snapshots", "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.UserAgent()
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	found, err := New(srv.Client(), srv.URL).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []discover.Company{
		{Slug: "faculty", Name: "Faculty", Board: discover.Board{Source: "wttj", Token: "Faculty"}},
		{Slug: "1password", Name: "1password", Board: discover.Board{Source: "wttj", Token: "1Password"}},
		{Slug: "acme-labs", Name: "Acme Labs", Board: discover.Board{Source: "wttj", Token: "acme-labs"}},
	}
	if diff := cmp.Diff(want, found.Companies); diff != "" {
		t.Errorf("Harvest companies (-want +got):\n%s", diff)
	}
	if gotUA != userAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, userAgent)
	}
}
