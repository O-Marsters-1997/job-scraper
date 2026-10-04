package yc_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/yc"
)

func TestHarvester_Harvest(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "all.json"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	var fetched []string
	get := func(_ context.Context, url string) ([]byte, error) {
		fetched = append(fetched, url)
		if url == "https://acme.co.uk" {
			return []byte(`<a href="https://boards.greenhouse.io/acme">Jobs</a>`), nil
		}
		return nil, fmt.Errorf("unexpected GET %s", url)
	}

	found, err := yc.New(srv.Client(), srv.URL, get).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := discover.Harvest{Companies: []discover.Company{
		{Slug: "acme-uk", Name: "Acme UK", Domain: "acme.co.uk", Board: discover.Board{Source: "greenhouse", Token: "acme"}},
	}}
	if diff := cmp.Diff(want, found); diff != "" {
		t.Errorf("Harvest (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"https://acme.co.uk"}, fetched); diff != "" {
		t.Errorf("Harvest fetched (-want +got):\n%s", diff)
	}
}

func TestHarvester_Harvest_unresolvedWebsiteIsSkipped(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "all.json"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	get := func(context.Context, string) ([]byte, error) { return []byte(`<p>hello</p>`), nil }

	found, err := yc.New(srv.Client(), srv.URL, get).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := discover.Harvest{Skipped: 1}
	if diff := cmp.Diff(want, found); diff != "" {
		t.Errorf("Harvest (-want +got):\n%s", diff)
	}
}
