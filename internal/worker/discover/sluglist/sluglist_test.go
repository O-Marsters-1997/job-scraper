package sluglist_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/sluglist"
)

func serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	t.Cleanup(srv.Close)
	return srv
}

func TestHarvester_Harvest(t *testing.T) {
	t.Parallel()
	srv := serve(t)
	lists := []sluglist.List{
		{Source: "workable", URL: srv.URL + "/workable.csv", Format: sluglist.CSV},
		{Source: "teamtailor", URL: srv.URL + "/teamtailor.csv", Format: sluglist.CSV},
		{Source: "greenhouse", URL: srv.URL + "/greenhouse.json", Format: sluglist.JSONSlugs},
		{Source: "lever", URL: srv.URL + "/missing.json", Format: sluglist.JSONSlugs},
	}

	got, err := sluglist.New(srv.Client(), lists).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := discover.Harvest{
		Boards: []discover.Board{
			{Source: "workable", Token: "acme"},
			{Source: "workable", Token: "slugonly"},
			{Source: "workable", Token: "urltoken"},
			{Source: "teamtailor", Token: "teaco"},
			{Source: "greenhouse", Token: "acme"},
		},
		Skipped: 4,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Harvest (-want +got):\n%s", diff)
	}
}

func TestHarvester_Harvest_allListsFailing(t *testing.T) {
	t.Parallel()
	srv := serve(t)
	lists := []sluglist.List{{Source: "lever", URL: srv.URL + "/missing.json", Format: sluglist.JSONSlugs}}

	if _, err := sluglist.New(srv.Client(), lists).Harvest(t.Context()); err == nil {
		t.Error("Harvest = nil error, want error when every list fails")
	}
}
