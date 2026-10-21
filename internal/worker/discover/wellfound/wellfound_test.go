package wellfound_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/wellfound"
)

type fakeVerified []string

func (f fakeVerified) ListVerifiedCompanySlugs(_ context.Context, slugs []string) ([]string, error) {
	var out []string
	for _, s := range slugs {
		if slices.Contains(f, s) {
			out = append(out, s)
		}
	}
	return out, nil
}

type listing struct {
	id, slug, company, ats string
}

func searchPage(t *testing.T, pageCount int, listings ...listing) []byte {
	t.Helper()
	data := map[string]any{}
	var startups []map[string]string
	for i, l := range listings {
		startupRef, listingRef := fmt.Sprintf("Startup:%d", i), fmt.Sprintf("Listing:%d", i)
		data[startupRef] = map[string]any{"name": l.company, "highlightedJobListings": []map[string]string{{"__ref": listingRef}}}
		data[listingRef] = map[string]any{"id": l.id, "slug": l.slug, "atsSource": l.ats}
		startups = append(startups, map[string]string{"__ref": startupRef})
	}
	data["ROOT_QUERY"] = map[string]any{"talent": map[string]any{
		"seoLandingPageJobSearchResults(x)": map[string]any{"pageCount": pageCount, "startups": startups},
	}}
	next, err := json.Marshal(map[string]any{"props": map[string]any{"pageProps": map[string]any{"apolloState": map[string]any{"data": data}}}})
	if err != nil {
		t.Fatal(err)
	}
	return []byte(`<html><script id="__NEXT_DATA__" type="application/json">` + string(next) + `</script></html>`)
}

func readTestdata(t *testing.T, file string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestHarvester_Harvest(t *testing.T) {
	t.Parallel()
	t.Run("returns ATS-backed unverified companies with a resolvable website", func(t *testing.T) {
		t.Parallel()
		pages := map[string][]byte{
			"https://wellfound.com/location/london": searchPage(t, 2,
				listing{"1", "swe", "AssemblyAI", "AtsIntegration::Greenhouse::Listing"},
				listing{"2", "native", "Native Co", ""},
				listing{"3", "known", "Verified Co", "AtsIntegration::Ashby::Listing"},
			),
			"https://wellfound.com/location/london?page=2": searchPage(t, 2,
				listing{"4", "ie", "Constructor.io", "AtsIntegration::Ashby::Listing"},
				listing{"5", "dev", "No Site Co", "AtsIntegration::Greenhouse::Listing"},
				listing{"6", "again", "AssemblyAI", "AtsIntegration::Greenhouse::Listing"},
			),
			"https://wellfound.com/jobs/1-swe": readTestdata(t, "detail_greenhouse.html"),
			"https://wellfound.com/jobs/4-ie":  readTestdata(t, "detail_ashby.html"),
			"https://wellfound.com/jobs/5-dev": []byte(`<html><body>no json-ld</body></html>`),
		}
		var requested []string
		getPages := func(_ context.Context, url string) ([]byte, error) {
			requested = append(requested, url)
			body, ok := pages[url]
			if !ok {
				return nil, fmt.Errorf("unexpected GET %s", url)
			}
			return body, nil
		}
		sites := map[string]string{
			"https://assemblyai.com": `<a href="https://boards.greenhouse.io/assemblyai">Jobs</a>`,
			"https://constructor.io": `<a href="https://jobs.ashbyhq.com/constructor">Jobs</a>`,
		}
		getSites := func(_ context.Context, url string) ([]byte, error) {
			body, ok := sites[url]
			if !ok {
				return nil, fmt.Errorf("unexpected GET %s", url)
			}
			return []byte(body), nil
		}

		h := wellfound.New(getPages, getSites, fakeVerified{"verified"}, []string{"london"})
		got, err := h.Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		want := discover.Harvest{
			Companies: []discover.Company{
				{Slug: "assemblyai", Name: "AssemblyAI", Domain: "assemblyai.com", Board: discover.Board{Source: "greenhouse", Token: "assemblyai"}},
				{Slug: "constructorio", Name: "Constructor.io", Domain: "constructor.io", Board: discover.Board{Source: "ashby", Token: "constructor"}},
			},
			Skipped: 1,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
		for _, url := range requested {
			if strings.Contains(url, "/jobs/3-") {
				t.Errorf("Harvest fetched detail page %s of an already verified company", url)
			}
		}
	})

	t.Run("keeps going when one location fails", func(t *testing.T) {
		t.Parallel()
		getPages := func(_ context.Context, url string) ([]byte, error) {
			if strings.Contains(url, "/location/london") {
				return nil, errors.New("blocked")
			}
			return searchPage(t, 1), nil
		}

		got, err := wellfound.New(getPages, nil, fakeVerified{}, []string{"london", "manchester"}).Harvest(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(discover.Harvest{}, got); diff != "" {
			t.Errorf("Harvest (-want +got):\n%s", diff)
		}
	})

	t.Run("fails when every location fails", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("blocked")
		getPages := func(context.Context, string) ([]byte, error) { return nil, boom }

		_, err := wellfound.New(getPages, nil, fakeVerified{}, wellfound.LocationPages).Harvest(t.Context())

		if !errors.Is(err, boom) {
			t.Errorf("Harvest error = %v, want it to wrap %v", err, boom)
		}
	})
}
