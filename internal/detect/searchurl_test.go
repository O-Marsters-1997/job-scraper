package detect_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/detect"
)

func TestParseSearchURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want detect.SearchURL
	}{
		{
			name: "linkedin search-results with tracking params",
			url:  "https://www.linkedin.com/jobs/search-results/?currentJobId=4012345678&keywords=golang%20engineer&origin=JOB_SEARCH_PAGE_JOB_FILTER&referralSearchId=abc%3D%3D&f_TPR=r604800&f_WT=2&geoId=101165590",
			want: detect.SearchURL{
				Source:  "linkedin",
				Value:   "golang engineer",
				Filters: map[string]string{"recency": "r604800", "arrangement": "2", "geo_id": "101165590"},
				Dropped: []string{"currentJobId", "origin", "referralSearchId"},
			},
		},
		{
			name: "linkedin later page",
			url:  "https://uk.linkedin.com/jobs/search/?keywords=platform&location=London&start=50&utm_source=share",
			want: detect.SearchURL{
				Source:  "linkedin",
				Value:   "platform",
				Filters: map[string]string{"location": "London"},
				Dropped: []string{"utm_source"},
			},
		},
		{
			name: "linkedin unknown param and invalid option are dropped",
			url:  "https://www.linkedin.com/jobs/search/?keywords=go&f_AL=true&f_TPR=r1&f_WT=1%2C2",
			want: detect.SearchURL{
				Source:  "linkedin",
				Value:   "go",
				Filters: map[string]string{},
				Dropped: []string{"f_AL", "f_TPR", "f_WT"},
			},
		},
		{
			name: "wis with paging",
			url:  "https://workinstartups.com/search?q=product+engineer&w=uk&p=3&per_page=50",
			want: detect.SearchURL{
				Source:  "wis",
				Value:   "product engineer",
				Filters: map[string]string{"region": "uk"},
				Dropped: []string{},
			},
		},
		{
			name: "indeed tracking params are dropped and filters named",
			url:  "https://uk.indeed.com/jobs?q=golang&l=London&vjk=abc123&from=searchOnDesktopSerp&start=10&fromage=7",
			want: detect.SearchURL{
				Source:  "indeed",
				Value:   "golang",
				Filters: map[string]string{"location": "London", "recency": "7"},
				Dropped: []string{"from", "vjk"},
			},
		},
	}
	t.Run("parses", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, ok := detect.ParseSearchURL(tt.url)
				if !ok {
					t.Fatalf("ParseSearchURL(%q) ok = false, want true", tt.url)
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("ParseSearchURL(%q) mismatch (-want +got):\n%s", tt.url, diff)
				}
			})
		}
	})

	t.Run("rejects", func(t *testing.T) {
		for _, raw := range []string{
			"",
			"not a url",
			"https://example.com/jobs/search/?keywords=go",
			"https://www.linkedin.com/jobs/view/1234567890",
			"https://www.linkedin.com/jobs/collections/recommended/",
			"https://www.indeed.com/viewjob?jk=abc",
			"https://remoteok.com/remote-golang-jobs",
		} {
			t.Run(raw, func(t *testing.T) {
				if got, ok := detect.ParseSearchURL(raw); ok {
					t.Errorf("ParseSearchURL(%q) = %+v, want not ok", raw, got)
				}
			})
		}
	})
}

func TestUnsupportedBoard(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"remoteok", "https://remoteok.com/remote-golang-jobs", "RemoteOK"},
		{"remotive", "https://remotive.com/remote-jobs/software-dev", "Remotive"},
		{"careers page", "https://example.com/careers", ""},
		{"not a url", "nonsense", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := detect.UnsupportedBoard(tt.url)
			if got != tt.want || ok != (tt.want != "") {
				t.Errorf("UnsupportedBoard(%q) = %q, %v, want %q", tt.url, got, ok, tt.want)
			}
		})
	}
}

func TestBuildSearchURL(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		value   string
		filters map[string]string
		want    string
	}{
		{
			name:   "linkedin keywords only",
			source: "linkedin", value: "go",
			want: "https://www.linkedin.com/jobs/search/?keywords=go",
		},
		{
			name:   "wis region",
			source: "wis", value: "product engineer", filters: map[string]string{"region": "uk"},
			want: "https://workinstartups.com/search?q=product+engineer&w=uk",
		},
		{
			name:   "indeed keywords and filters",
			source: "indeed", value: "go", filters: map[string]string{"location": "London", "radius": "25"},
			want: "https://uk.indeed.com/jobs?l=London&q=go&radius=25",
		},
		{name: "source without a search page", source: "greenhouse", value: "acme", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detect.BuildSearchURL(tt.source, tt.value, tt.filters); got != tt.want {
				t.Errorf("BuildSearchURL(%q, %q, %v) = %q, want %q", tt.source, tt.value, tt.filters, got, tt.want)
			}
		})
	}

	t.Run("round trips through ParseSearchURL", func(t *testing.T) {
		linkedin := map[string]string{
			"location": "London", "company_id": "1337", "recency": "r86400", "arrangement": "3",
			"experience": "4", "job_type": "F", "geo_id": "101165590", "distance": "25", "salary_band": "5",
		}
		tests := []struct {
			name    string
			source  string
			value   string
			filters map[string]string
		}{
			{"linkedin every filter", "linkedin", "golang engineer", linkedin},
			{"linkedin keywords only", "linkedin", "go", map[string]string{}},
			{"wis region", "wis", "product engineer", map[string]string{"region": "uk"}},
			{"wis keywords only", "wis", "go", map[string]string{}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				built := detect.BuildSearchURL(tt.source, tt.value, tt.filters)
				parsed, ok := detect.ParseSearchURL(built)
				if !ok {
					t.Fatalf("ParseSearchURL(%q) ok = false", built)
				}
				want := detect.SearchURL{Source: tt.source, Value: tt.value, Filters: tt.filters, Dropped: []string{}}
				if diff := cmp.Diff(want, parsed); diff != "" {
					t.Errorf("Parse(Build(...)) mismatch (-want +got):\n%s", diff)
				}
				if again := detect.BuildSearchURL(parsed.Source, parsed.Value, parsed.Filters); again != built {
					t.Errorf("Build(Parse(%q)) = %q", built, again)
				}
			})
		}

		t.Run("indeed", func(t *testing.T) {
			raw := "https://uk.indeed.com/jobs?fromage=7&l=London&q=golang&radius=25"
			parsed, ok := detect.ParseSearchURL(raw)
			if !ok {
				t.Fatal("ok = false")
			}
			if got := detect.BuildSearchURL(parsed.Source, parsed.Value, parsed.Filters); got != raw {
				t.Errorf("Build(Parse(%q)) = %q", raw, got)
			}
		})
	})
}

func FuzzParseSearchURL(f *testing.F) {
	for _, seed := range []string{
		"",
		"not a url",
		"https://www.linkedin.com/jobs/search-results/?currentJobId=4012345678&keywords=golang%20engineer&f_TPR=r604800&f_WT=2&geoId=101165590",
		"https://uk.linkedin.com/jobs/search/?keywords=platform&location=London&start=50&utm_source=share",
		"https://workinstartups.com/search?q=product+engineer&w=uk&p=3&per_page=50",
		"https://uk.indeed.com/jobs?q=golang&l=London&vjk=abc123&start=10&fromage=7",
		"https://www.linkedin.com/jobs/view/1234567890",
		"https://remoteok.com/remote-golang-jobs",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		p, ok := detect.ParseSearchURL(raw)
		if !ok {
			return
		}
		built := detect.BuildSearchURL(p.Source, p.Value, p.Filters)
		again, ok := detect.ParseSearchURL(built)
		if !ok {
			t.Fatalf("ParseSearchURL(BuildSearchURL(%+v)) ok = false, built %q", p, built)
		}
		if diff := cmp.Diff(p, again, cmpopts.IgnoreFields(detect.SearchURL{}, "Dropped")); diff != "" {
			t.Errorf("round trip of %q (-want +got):\n%s", raw, diff)
		}
	})
}
