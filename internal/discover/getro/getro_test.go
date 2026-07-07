package getro

import (
	"os"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/discover"
)

func TestParseGetro(t *testing.T) {
	body, err := os.ReadFile("testdata/jobsinvc_jobs.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	companies, err := parseGetro(body)
	if err != nil {
		t.Fatalf("parseGetro: %v", err)
	}

	// The fixture (a real jobsinvc.getro.com/jobs response) has 20 jobs
	// across 11 unique organizations; OMERS Ventures appears on 2 of them
	// and must be deduplicated down to a single Company.
	if len(companies) != 11 {
		t.Fatalf("got %d companies, want 11: %+v", len(companies), companies)
	}

	byName := make(map[string]discover.Company)
	for _, c := range companies {
		if _, dup := byName[c.Name]; dup {
			t.Fatalf("%q appears more than once in results: %+v", c.Name, companies)
		}
		byName[c.Name] = c
	}

	// StepStone Group's job links to boards.greenhouse.io/stepstone/... —
	// should resolve to a greenhouse ATS source+token.
	got, ok := byName["StepStone Group"]
	if !ok {
		t.Fatalf("want StepStone Group in results, got %+v", byName)
	}
	if got.ATSSource != "greenhouse" || got.ATSToken != "stepstone" {
		t.Errorf("got ats=%q token=%q, want greenhouse/stepstone", got.ATSSource, got.ATSToken)
	}

	// Aligned Climate Capital's job links to linkedin.com, an aggregator
	// ResolveBoard can't attribute to any ATS — name-only is expected.
	got, ok = byName["Aligned Climate Capital"]
	if !ok {
		t.Fatalf("want Aligned Climate Capital in results, got %+v", byName)
	}
	if got.ATSSource != "" || got.ATSToken != "" {
		t.Errorf("got ats=%q token=%q, want empty for an unresolvable LinkedIn URL", got.ATSSource, got.ATSToken)
	}
}

func TestParseGetro_NoNextData(t *testing.T) {
	if _, err := parseGetro([]byte(`<html><body>no data here</body></html>`)); err == nil {
		t.Error("want error when __NEXT_DATA__ script is missing, got nil")
	}
}

func TestParseGetro_InvalidJSON(t *testing.T) {
	body := []byte(`<script id="__NEXT_DATA__" type="application/json">not json</script>`)
	if _, err := parseGetro(body); err == nil {
		t.Error("want error for invalid JSON in __NEXT_DATA__, got nil")
	}
}
