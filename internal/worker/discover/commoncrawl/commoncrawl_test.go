package commoncrawl_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/commoncrawl"
)

func fixtureKind(pattern string) string {
	for _, kind := range []struct{ host, name string }{
		{"ashby", "ashby"},
		{"job-boards.eu.greenhouse", "greenhouse_eu"},
		{"job-boards.greenhouse", "greenhouse_jobboards"},
		{"greenhouse", "greenhouse"},
		{"lever", "lever"},
		{"workable", "workable"},
		{"recruitee", "recruitee"},
		{"personio", "personio"},
	} {
		if strings.Contains(pattern, kind.host) {
			return kind.name
		}
	}
	return ""
}

func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := ""
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/collinfo.json":
			file = "collinfo.json"
		case r.URL.Path != "/CC-MAIN-2099-01-index":
			http.NotFound(w, r)
			return
		default:
			kind := fixtureKind(q.Get("url"))
			if q.Get("showNumPages") != "" {
				file = "pages_" + kind + ".json"
			} else {
				file = kind + "_" + q.Get("page") + ".ndjson"
			}
		}
		body, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(strings.ReplaceAll(string(body), "@BASE@", srv.URL)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHarvest_ResolvesEveryATSPatternAndSkipsNonBoards(t *testing.T) {
	srv := fixtureServer(t)

	got, err := commoncrawl.New(srv.Client(), srv.URL+"/collinfo.json").Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := discover.Harvest{
		Boards: []discover.Board{
			{Source: "ashby", Token: "acme"},
			{Source: "ashby", Token: "beta-co"},
			{Source: "greenhouse", Token: "gamma"},
			{Source: "greenhouse", Token: "delta"},
			{Source: "greenhouse", Token: "epsilon"},
			{Source: "greenhouse", Token: "zeta"},
			{Source: "greenhouse", Token: "eta"},
			{Source: "greenhouse", Token: "theta"},
			{Source: "lever", Token: "iota"},
			{Source: "workable", Token: "kappa"},
			{Source: "recruitee", Token: "lambda"},
			{Source: "personio", Token: "mu"},
		},
		Skipped: 8,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Harvest() (-want +got):\n%s", diff)
	}
}

func TestHarvest_ErrorsWhenIndexFetchFails(t *testing.T) {
	srv := fixtureServer(t)
	if _, err := commoncrawl.New(srv.Client(), srv.URL+"/missing.json").Harvest(t.Context()); err == nil {
		t.Error("Harvest() = nil error, want one")
	}
}
