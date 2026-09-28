package getro_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/getro"
)

func serveBoard(t *testing.T, body []byte) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jobs" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestHarvestParsesBoardCompanies(t *testing.T) {
	fixture, err := os.ReadFile("testdata/jobsinvc_jobs.html")
	if err != nil {
		t.Fatal(err)
	}
	base := serveBoard(t, fixture)

	companies, err := getro.New().WithBoards([]string{base + "/jobs"}).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	byName := make(map[string]discover.Company)
	for _, c := range companies {
		if _, dup := byName[c.Name]; dup {
			t.Fatalf("%q appears more than once in results: %+v", c.Name, companies)
		}
		byName[c.Name] = c
	}
	if len(companies) != 11 {
		t.Fatalf("got %d companies, want 11: %+v", len(companies), companies)
	}
	if got := byName["StepStone Group"]; got.ATSSource != "greenhouse" || got.ATSToken != "stepstone" {
		t.Errorf("StepStone Group = %+v, want greenhouse/stepstone", got)
	}
	got, ok := byName["Aligned Climate Capital"]
	if !ok {
		t.Fatalf("want Aligned Climate Capital in results, got %+v", byName)
	}
	if got.ATSSource != "" || got.ATSToken != "" {
		t.Errorf("Aligned Climate Capital = %+v, want name only for an unresolvable LinkedIn URL", got)
	}
}

func TestHarvestSkipsBrokenBoards(t *testing.T) {
	for name, page := range map[string]string{
		"no __NEXT_DATA__": `<html><body>no data here</body></html>`,
		"invalid JSON":     `<script id="__NEXT_DATA__" type="application/json">not json</script>`,
	} {
		t.Run(name, func(t *testing.T) {
			base := serveBoard(t, []byte(page))
			companies, err := getro.New().WithBoards([]string{base + "/jobs", base + "/missing"}).Harvest(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if len(companies) != 0 {
				t.Fatalf("got %+v, want no companies", companies)
			}
		})
	}
}
