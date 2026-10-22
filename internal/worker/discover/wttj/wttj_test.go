package wttj_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
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

var aliasedID = regexp.MustCompile(`(j\d+): publicJob\(externalId: "([^"]+)"`)

type byID struct {
	jobs    map[string]json.RawMessage
	blocked bool
	queries int
}

func (a *byID) RoundTrip(r *http.Request) (*http.Response, error) {
	a.queries++
	rec := httptest.NewRecorder()
	if a.blocked {
		rec.WriteHeader(http.StatusTooManyRequests)
		return rec.Result(), nil
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var req struct{ Query string }
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	data := map[string]json.RawMessage{}
	for _, m := range aliasedID.FindAllStringSubmatch(req.Query, -1) {
		if job, ok := a.jobs[m[2]]; ok {
			data[m[1]] = job
		} else {
			data[m[1]] = json.RawMessage("null")
		}
	}
	if err := json.NewEncoder(rec).Encode(map[string]any{"data": data}); err != nil {
		return nil, err
	}
	return rec.Result(), nil
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("snapshots", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func listingsByID(t *testing.T) *byID {
	t.Helper()
	var f struct {
		Data map[string]json.RawMessage
	}
	if err := json.Unmarshal([]byte(fixture(t, "listings.json")), &f); err != nil {
		t.Fatal(err)
	}
	api := &byID{jobs: map[string]json.RawMessage{}}
	for _, raw := range f.Data {
		var j struct{ ExternalID string }
		if err := json.Unmarshal(raw, &j); err != nil {
			t.Fatal(err)
		}
		if j.ExternalID != "" {
			api.jobs[j.ExternalID] = raw
		}
	}
	return api
}

type recordingExporter struct {
	jobs []dto.Job
	err  error
}

func (e *recordingExporter) Export(_ context.Context, job dto.Job) error {
	if e.err != nil {
		return e.err
	}
	e.jobs = append(e.jobs, job)
	return nil
}

func harvester(t *testing.T, sitemap string, api http.RoundTripper, seen wttj.SeenStore, exp wttj.Exporter) *wttj.Harvester {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sitemap))
	}))
	t.Cleanup(srv.Close)
	client := wttjsource.NewClient(0)
	client.Client().Transport = api
	return wttj.New(srv.Client(), srv.URL, client, seen, exp)
}

func seenStore(t *testing.T, ids ...string) *jobsearchtest.FakeStore {
	t.Helper()
	st := jobsearchtest.NewFakeStore()
	if err := st.RecordHarvestSeenIDs(t.Context(), "wttj", "backfill", ids); err != nil {
		t.Fatal(err)
	}
	return st
}

func seenIDs(t *testing.T, st *jobsearchtest.FakeStore) []string {
	t.Helper()
	ids, err := st.HarvestSeenIDs(t.Context(), "wttj")
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(ids)
	return ids
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
	t.Run("first run records every sitemap job as backfill and exports nothing", func(t *testing.T) {
		t.Parallel()
		api, exp, st := listingsByID(t), &recordingExporter{}, jobsearchtest.NewFakeStore()
		got, err := harvester(t, fixture(t, "sitemap.xml"), api, st, exp).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(discover.Harvest{Skipped: 1}, got); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
		wantIDs := []string{"ats1", "ats2", "gone1", "mkt1", "noats1", "us1"}
		if diff := cmp.Diff(wantIDs, seenIDs(t, st)); diff != "" {
			t.Errorf("seen IDs (-want +got):\n%s", diff)
		}
		if api.queries != 0 || len(exp.jobs) != 0 {
			t.Errorf("backfill made %d lookups and %d exports, want none", api.queries, len(exp.jobs))
		}
	})

	t.Run("new IDs are exported, handed to their ATS, or recorded as ineligible, and departed IDs are retired", func(t *testing.T) {
		t.Parallel()
		exp, st := &recordingExporter{}, seenStore(t, "departed")
		got, err := harvester(t, fixture(t, "sitemap.xml"), listingsByID(t), st, exp).Harvest(t.Context())
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
		var exported []string
		for _, j := range exp.jobs {
			exported = append(exported, j.ProviderPostingID)
		}
		if diff := cmp.Diff([]string{"noats1"}, exported); diff != "" {
			t.Errorf("exported (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"gone1", "mkt1", "noats1", "us1"}, seenIDs(t, st)); diff != "" {
			t.Errorf("seen IDs, ATS-backed left unrecorded and departed removed (-want +got):\n%s", diff)
		}
	})

	t.Run("known IDs are not looked up again", func(t *testing.T) {
		t.Parallel()
		api, exp := listingsByID(t), &recordingExporter{}
		st := seenStore(t, "ats1", "ats2", "gone1", "mkt1", "noats1", "us1")
		got, err := harvester(t, fixture(t, "sitemap.xml"), api, st, exp).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(discover.Harvest{Skipped: 1}, got); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
		if api.queries != 0 {
			t.Errorf("made %d lookups, want 0", api.queries)
		}
	})

	t.Run("a deferred lookup leaves new IDs unrecorded", func(t *testing.T) {
		t.Parallel()
		api := listingsByID(t)
		api.blocked = true
		st := seenStore(t, "departed")
		if _, err := harvester(t, fixture(t, "sitemap.xml"), api, st, &recordingExporter{}).Harvest(t.Context()); !errors.Is(err, sources.ErrDeferred) {
			t.Errorf("Harvest error = %v, want ErrDeferred", err)
		}
		if diff := cmp.Diff([]string{"departed"}, seenIDs(t, st)); diff != "" {
			t.Errorf("seen IDs (-want +got):\n%s", diff)
		}
	})

	t.Run("a failed export leaves that ID unrecorded for the next run", func(t *testing.T) {
		t.Parallel()
		st := seenStore(t, "departed")
		exp := &recordingExporter{err: errors.New("ingest down")}
		if _, err := harvester(t, fixture(t, "sitemap.xml"), listingsByID(t), st, exp).Harvest(t.Context()); err != nil {
			t.Fatal(err)
		}
		if slices.Contains(seenIDs(t, st), "noats1") {
			t.Error("noats1 recorded despite failed export")
		}
	})

	t.Run("a block after some listings were fetched keeps them", func(t *testing.T) {
		t.Parallel()
		eligible := ukEngineeringJobs(100)
		api := &graphql{responses: []func(http.ResponseWriter){body(eligible), body(eligible), status(http.StatusTooManyRequests)}}
		got, err := harvester(t, sitemapOf(300), api, seenStore(t, "old"), &recordingExporter{}).Harvest(t.Context())
		if err != nil {
			t.Fatalf("Harvest error = %v, want the fetched listings kept", err)
		}
		if diff := cmp.Diff([]discover.Board{{Source: "ashby", Token: "acme"}}, got.Boards); diff != "" {
			t.Errorf("Harvest boards (-want +got):\n%s", diff)
		}
	})
}
