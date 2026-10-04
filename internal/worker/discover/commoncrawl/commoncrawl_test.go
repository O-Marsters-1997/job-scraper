package commoncrawl_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

func crawlServer(t *testing.T, ids []string, handler func(w http.ResponseWriter, id, pattern string)) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collinfo.json" {
			var entries []string
			for _, id := range ids {
				entries = append(entries, fmt.Sprintf(`{"id":%q,"cdx-api":"%s/%s-index"}`, id, srv.URL, id))
			}
			_, _ = fmt.Fprintf(w, "[%s]", strings.Join(entries, ","))
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), "-index")
		pattern := r.URL.Query().Get("url")
		if r.URL.Query().Get("showNumPages") != "" {
			if !strings.HasPrefix(pattern, "jobs.lever.co") {
				_, _ = w.Write([]byte(`{"pages":0}`))
				return
			}
			handler(w, id, "pages")
			return
		}
		handler(w, id, pattern)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func leverRow(token string) string {
	return fmt.Sprintf(`{"url":"https://jobs.lever.co/%s"}`+"\n", token)
}

func TestHarvest_RetriesTransientServerErrors(t *testing.T) {
	var cdxCalls atomic.Int32
	srv := crawlServer(t, []string{"CC-MAIN-2099-01"}, func(w http.ResponseWriter, _, pattern string) {
		if pattern == "pages" {
			if cdxCalls.Add(1) == 1 {
				http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
				return
			}
			_, _ = w.Write([]byte(`{"pages":1}`))
			return
		}
		_, _ = w.Write([]byte(leverRow("acme")))
	})

	got, err := commoncrawl.New(srv.Client(), srv.URL+"/collinfo.json", commoncrawl.WithBackoff(time.Millisecond)).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := discover.Harvest{Boards: []discover.Board{{Source: "lever", Token: "acme"}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Harvest() (-want +got):\n%s", diff)
	}
}

func TestHarvest_CountsAlwaysFailingPatternAsSkipped(t *testing.T) {
	var cdxCalls atomic.Int32
	srv := crawlServer(t, []string{"CC-MAIN-2099-01"}, func(w http.ResponseWriter, _, pattern string) {
		if pattern == "pages" {
			cdxCalls.Add(1)
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		}
	})

	got, err := commoncrawl.New(srv.Client(), srv.URL+"/collinfo.json", commoncrawl.WithBackoff(time.Millisecond)).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.Skipped != 1 || len(got.Boards) != 0 {
		t.Errorf("Harvest() = %+v, want 1 skipped and no boards", got)
	}
	if n := cdxCalls.Load(); n != 4 {
		t.Errorf("failing pattern fetched %d times, want 4 attempts", n)
	}
}

func TestHarvest_DoesNotRetryClientErrors(t *testing.T) {
	var cdxCalls atomic.Int32
	srv := crawlServer(t, []string{"CC-MAIN-2099-01"}, func(w http.ResponseWriter, _, pattern string) {
		if pattern == "pages" {
			cdxCalls.Add(1)
			http.NotFound(w, nil)
		}
	})

	if _, err := commoncrawl.New(srv.Client(), srv.URL+"/collinfo.json", commoncrawl.WithBackoff(time.Millisecond)).Harvest(t.Context()); err != nil {
		t.Fatal(err)
	}

	if n := cdxCalls.Load(); n != 1 {
		t.Errorf("404 fetched %d times, want 1", n)
	}
}

func TestHarvest_UnionsNewestThreeCrawls(t *testing.T) {
	srv := crawlServer(t, []string{"c4", "c3", "c2", "c1"}, func(w http.ResponseWriter, id, pattern string) {
		switch {
		case pattern == "pages":
			_, _ = w.Write([]byte(`{"pages":1}`))
		default:
			_, _ = w.Write([]byte(leverRow("shared") + leverRow("only-"+id)))
		}
	})

	got, err := commoncrawl.New(srv.Client(), srv.URL+"/collinfo.json").Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []discover.Board{
		{Source: "lever", Token: "shared"},
		{Source: "lever", Token: "only-c4"},
		{Source: "lever", Token: "only-c3"},
		{Source: "lever", Token: "only-c2"},
	}
	if diff := cmp.Diff(want, got.Boards); diff != "" {
		t.Errorf("Harvest().Boards (-want +got):\n%s", diff)
	}
}

func TestHarvest_ErrorsWhenEveryPatternFails(t *testing.T) {
	srv := crawlServer(t, []string{"CC-MAIN-2099-01"}, func(http.ResponseWriter, string, string) {})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collinfo.json" {
			_, _ = fmt.Fprintf(w, `[{"cdx-api":"%s/x-index"}]`, srv.URL)
			return
		}
		http.Error(w, "down", http.StatusServiceUnavailable)
	})

	_, err := commoncrawl.New(srv.Client(), srv.URL+"/collinfo.json", commoncrawl.WithBackoff(time.Millisecond)).Harvest(t.Context())
	if err == nil {
		t.Error("Harvest() = nil error, want one")
	}
}
