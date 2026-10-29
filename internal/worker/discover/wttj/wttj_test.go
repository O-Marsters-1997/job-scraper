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
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/wttj"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	wttjsource "github.com/ollymarsters/job-scraper/internal/worker/sources/wttj"
)

var now = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

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

var (
	aliasedID      = regexp.MustCompile(`(j\d+): publicJob\(externalId: "([^"]+)"`)
	aliasedCompany = regexp.MustCompile(`(c\d+): company\(urlSafeName: "([^"]+)"`)
)

type byID struct {
	jobs        map[string]json.RawMessage
	counts      map[string]int
	companyJobs map[string][]string
	blocked     bool
	lookups     int
}

func (a *byID) RoundTrip(r *http.Request) (*http.Response, error) {
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
	data := map[string]any{}
	if strings.Contains(req.Query, "companies(") {
		companies := []map[string]any{}
		for name, n := range a.counts {
			companies = append(companies, map[string]any{"urlSafeName": name, "numberLiveJobs": n})
		}
		data["companies"] = companies
	}
	for _, m := range aliasedCompany.FindAllStringSubmatch(req.Query, -1) {
		ids, ok := a.companyJobs[m[2]]
		if !ok {
			data[m[1]] = nil
			continue
		}
		liveJobs := []map[string]string{}
		for _, id := range ids {
			liveJobs = append(liveJobs, map[string]string{"externalId": id})
		}
		data[m[1]] = map[string]any{"liveJobs": liveJobs}
	}
	for _, m := range aliasedID.FindAllStringSubmatch(req.Query, -1) {
		a.lookups++
		if job, ok := a.jobs[m[2]]; ok {
			data[m[1]] = job
		} else {
			data[m[1]] = nil
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

func (e *recordingExporter) ids() []string {
	var ids []string
	for _, j := range e.jobs {
		ids = append(ids, j.ProviderPostingID)
	}
	return ids
}

func harvester(t *testing.T, sitemap string, api http.RoundTripper, seen wttj.SeenStore, exp wttj.Exporter) *wttj.Harvester {
	t.Helper()
	h, _ := harvesterAt(t, func() time.Time { return now }, api, seen, exp, sitemap)
	return h
}

func harvesterAt(t *testing.T, clock func() time.Time, api http.RoundTripper, seen wttj.SeenStore, exp wttj.Exporter, sitemapsInTurn ...string) (h *wttj.Harvester, sitemapsFetched func() int) {
	t.Helper()
	var mu sync.Mutex
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		sitemap := sitemapsInTurn[min(hits, len(sitemapsInTurn)-1)]
		hits++
		mu.Unlock()
		_, _ = w.Write([]byte(sitemap))
	}))
	t.Cleanup(srv.Close)
	client := wttjsource.NewClient(0)
	client.Client().Transport = api
	sitemapsFetched = func() int {
		mu.Lock()
		defer mu.Unlock()
		return hits
	}
	return wttj.New(srv.Client(), srv.URL, client, seen, exp, clock), sitemapsFetched
}

func seenStore(t *testing.T, ids ...string) *jobsearchtest.FakeStore {
	t.Helper()
	st := jobsearchtest.NewFakeStore()
	if err := st.RecordHarvestSeenIDs(t.Context(), "wttj", "backfill", ids); err != nil {
		t.Fatal(err)
	}
	return st
}

var sitemapIDs = []string{"ats1", "ats2", "dead1", "gone1", "mkt1", "noats1", "old1", "us1"}

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
			"live":                true,
			"firstLiveAt":         "2026-10-28T09:00:00",
			"locationPreferences": []map[string]string{{"location": "LONDON"}},
			"subFunction":         map[string]string{"value": "Backend Engineer"},
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
		if diff := cmp.Diff(sitemapIDs, seenIDs(t, st)); diff != "" {
			t.Errorf("seen IDs (-want +got):\n%s", diff)
		}
		if api.lookups != 0 || len(exp.jobs) != 0 {
			t.Errorf("backfill made %d lookups and %d exports, want none", api.lookups, len(exp.jobs))
		}
	})

	t.Run("new IDs are exported, handed to their ATS, or recorded as ineligible, and absent IDs are kept", func(t *testing.T) {
		t.Parallel()
		exp, st := &recordingExporter{}, seenStore(t, "departed")
		got, err := harvester(t, fixture(t, "sitemap.xml"), listingsByID(t), st, exp).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		want := discover.Harvest{
			Checks: []discover.Check{
				{ID: "ats1", Board: discover.Board{Source: "ashby", Token: "acme", ListingURL: "https://app.welcometothejungle.com/jobs/ats1", ApplyURL: "https://jobs.ashbyhq.com/acme/1"}},
				{ID: "ats2", Board: discover.Board{Source: "ashby", Token: "acme", ListingURL: "https://app.welcometothejungle.com/jobs/ats2", ApplyURL: "https://jobs.ashbyhq.com/acme/2"}},
			},
			Companies: []discover.Company{
				{Slug: "steno-labs", Name: "Steno Labs", Board: discover.Board{Source: "wttj", Token: "Steno-Labs"}},
			},
			Skipped: 1,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"noats1"}, exp.ids()); diff != "" {
			t.Errorf("exported, without the dead and the 90-day-old listing (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"dead1", "departed", "gone1", "mkt1", "noats1", "old1", "us1"}, seenIDs(t, st)); diff != "" {
			t.Errorf("seen IDs, ATS-backed left unrecorded (-want +got):\n%s", diff)
		}
		if closed := st.HarvestClosedIDs("wttj"); len(closed) != 0 {
			t.Errorf("closed %v after one missing sitemap, want none", closed)
		}
	})

	t.Run("known IDs are not looked up again", func(t *testing.T) {
		t.Parallel()
		api, exp := listingsByID(t), &recordingExporter{}
		got, err := harvester(t, fixture(t, "sitemap.xml"), api, seenStore(t, sitemapIDs...), exp).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(discover.Harvest{Skipped: 1}, got); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
		if api.lookups != 0 {
			t.Errorf("made %d lookups, want 0", api.lookups)
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

	t.Run("an empty sitemap closes nothing", func(t *testing.T) {
		t.Parallel()
		st := seenStore(t, "a", "b", "c")
		nextMonth := func() time.Time { return time.Now().Add(30 * 24 * time.Hour) }
		h, _ := harvesterAt(t, nextMonth, listingsByID(t), st, &recordingExporter{}, sitemapOf(0))
		if _, err := h.Harvest(t.Context()); err != nil {
			t.Fatal(err)
		}
		if closed := st.HarvestClosedIDs("wttj"); len(closed) != 0 {
			t.Errorf("closed %v after an empty sitemap, want none", closed)
		}
	})

	t.Run("an ID absent from the sitemap for over a week has its job closed", func(t *testing.T) {
		t.Parallel()
		st := seenStore(t, append([]string{"departed"}, sitemapIDs...)...)
		nextWeek := func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
		h, _ := harvesterAt(t, nextWeek, listingsByID(t), st, &recordingExporter{}, fixture(t, "sitemap.xml"))
		if _, err := h.Harvest(t.Context()); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"departed"}, st.HarvestClosedIDs("wttj")); diff != "" {
			t.Errorf("closed IDs (-want +got):\n%s", diff)
		}
		if !slices.Contains(seenIDs(t, st), "departed") {
			t.Error("departed dropped from seen IDs, want it kept so a return is not new")
		}
	})

	t.Run("a blocked API still closes jobs absent from the sitemap for over a week", func(t *testing.T) {
		t.Parallel()
		api := listingsByID(t)
		api.blocked = true
		st := seenStore(t, "departed", "ats1")
		nextWeek := func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
		h, _ := harvesterAt(t, nextWeek, api, st, &recordingExporter{}, fixture(t, "sitemap.xml"))
		if _, err := h.Harvest(t.Context()); !errors.Is(err, sources.ErrDeferred) {
			t.Fatalf("Harvest error = %v, want ErrDeferred", err)
		}
		if diff := cmp.Diff([]string{"departed"}, st.HarvestClosedIDs("wttj")); diff != "" {
			t.Errorf("closed IDs (-want +got):\n%s", diff)
		}
	})

	t.Run("an ID that drops out of the sitemap and comes back is neither closed nor exported again", func(t *testing.T) {
		t.Parallel()
		full := fixture(t, "sitemap.xml")
		without := strings.Replace(full, "<url><loc>https://app.welcometothejungle.com/jobs/noats1</loc></url>", "", 1)
		st, exp := seenStore(t, sitemapIDs...), &recordingExporter{}
		at := time.Now()
		h, fetched := harvesterAt(t, func() time.Time { return at }, listingsByID(t), st, exp, without, full)
		for range 2 {
			if _, err := h.Harvest(t.Context()); err != nil {
				t.Fatal(err)
			}
			at = at.Add(24 * time.Hour)
		}
		if fetched() != 2 {
			t.Fatalf("fetched %d sitemaps, want 2", fetched())
		}
		if closed := st.HarvestClosedIDs("wttj"); len(closed) != 0 {
			t.Errorf("closed %v, want none", closed)
		}
		if len(exp.jobs) != 0 {
			t.Errorf("exported %v, want none", exp.ids())
		}
	})

	t.Run("a company whose live count rose has its unseen jobs exported before the next sitemap", func(t *testing.T) {
		t.Parallel()
		st, exp := seenStore(t, sitemapIDs...), &recordingExporter{}
		if err := st.RecordHarvestCompanyCounts(t.Context(), "wttj", map[string]int{"Steno-Labs": 1, "Acme": 2, "Gone-Co": 3}); err != nil {
			t.Fatal(err)
		}
		api := listingsByID(t)
		api.counts = map[string]int{"Steno-Labs": 1, "Acme": 2, "Gone-Co": 3}
		api.companyJobs = map[string][]string{"Steno-Labs": {"noats1", "new1"}, "Acme": {"ats1", "ats2"}}
		at := now
		h, fetched := harvesterAt(t, func() time.Time { return at }, api, st, exp, fixture(t, "sitemap.xml"))
		if _, err := h.Harvest(t.Context()); err != nil {
			t.Fatal(err)
		}

		at = at.Add(15 * time.Minute)
		api.counts = map[string]int{"Steno-Labs": 2, "Acme": 2, "*": 4}
		got, err := h.Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if fetched() != 1 {
			t.Errorf("fetched %d sitemaps, want 1 within a day", fetched())
		}
		if diff := cmp.Diff([]string{"new1"}, exp.ids()); diff != "" {
			t.Errorf("exported (-want +got):\n%s", diff)
		}
		wantCompanies := []discover.Company{{Slug: "steno-labs", Name: "Steno Labs", Board: discover.Board{Source: "wttj", Token: "Steno-Labs"}}}
		if diff := cmp.Diff(wantCompanies, got.Companies); diff != "" {
			t.Errorf("Harvest companies (-want +got):\n%s", diff)
		}
		counts, err := st.HarvestCompanyCounts(t.Context(), "wttj")
		if err != nil {
			t.Fatal(err)
		}
		wantCounts := map[string]int{"Steno-Labs": 2, "Acme": 2, "*": 4, "Gone-Co": 3}
		if diff := cmp.Diff(wantCounts, counts); diff != "" {
			t.Errorf("stored counts (-want +got):\n%s", diff)
		}
	})

	t.Run("the sitemap is fetched again once a day has passed", func(t *testing.T) {
		t.Parallel()
		at := now
		h, fetched := harvesterAt(t, func() time.Time { return at }, listingsByID(t), seenStore(t, sitemapIDs...), &recordingExporter{}, fixture(t, "sitemap.xml"))
		for _, step := range []time.Duration{0, 15 * time.Minute, 24 * time.Hour} {
			at = at.Add(step)
			if _, err := h.Harvest(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
		if fetched() != 2 {
			t.Errorf("fetched %d sitemaps over a day and a quarter hour, want 2", fetched())
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

	t.Run("RecordChecks records the published checks so they are not handled again", func(t *testing.T) {
		t.Parallel()
		st := seenStore(t, "departed")
		h := harvester(t, fixture(t, "sitemap.xml"), listingsByID(t), st, &recordingExporter{})
		var _ discover.CheckRecorder = h
		if err := h.RecordChecks(t.Context(), []string{"ats1", "ats2"}); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"ats1", "ats2", "departed"}, seenIDs(t, st)); diff != "" {
			t.Errorf("seen IDs (-want +got):\n%s", diff)
		}
		got, err := h.Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Checks) != 0 {
			t.Errorf("Harvest checks = %+v, want none for recorded IDs", got.Checks)
		}
	})

	t.Run("a block after some listings were fetched keeps them", func(t *testing.T) {
		t.Parallel()
		eligible := ukEngineeringJobs(100)
		counts := `{"data":{"companies":[{"urlSafeName":"Acme","numberLiveJobs":5},{"urlSafeName":"New-Co","numberLiveJobs":0}]}}`
		api := &graphql{responses: []func(http.ResponseWriter){body(counts), body(eligible), body(eligible), status(http.StatusTooManyRequests)}}
		st := seenStore(t, "old")
		if err := st.RecordHarvestCompanyCounts(t.Context(), "wttj", map[string]int{"Acme": 5}); err != nil {
			t.Fatal(err)
		}
		h, fetched := harvesterAt(t, func() time.Time { return now }, api, st, &recordingExporter{}, sitemapOf(300))
		got, err := h.Harvest(t.Context())
		if err != nil {
			t.Fatalf("Harvest error = %v, want the fetched listings kept", err)
		}
		if len(got.Checks) == 0 {
			t.Error("Harvest checks empty, want the fetched listings kept")
		}
		stored, err := st.HarvestCompanyCounts(t.Context(), "wttj")
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(map[string]int{"Acme": 5}, stored); diff != "" {
			t.Errorf("stored counts after an unfinished run, want them left for a retry (-want +got):\n%s", diff)
		}
		if _, err := h.Harvest(t.Context()); !errors.Is(err, sources.ErrDeferred) {
			t.Errorf("second Harvest error = %v, want ErrDeferred while still blocked", err)
		}
		if fetched() != 2 {
			t.Errorf("fetched %d sitemaps, want the unfinished one fetched again on the next run", fetched())
		}
	})
}
