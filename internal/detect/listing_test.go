package detect_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/detect"
)

func TestResolveListing(t *testing.T) {
	t.Parallel()

	const li = "https://www.linkedin.com/jobs/view/3912345678"
	tests := []struct {
		name          string
		url           string
		source, canon string
		ok            bool
	}{
		{"linkedin plain", "https://www.linkedin.com/jobs/view/3912345678", "linkedin", li, true},
		{"linkedin plain with query", "https://www.linkedin.com/jobs/view/3912345678/?trk=x", "linkedin", li, true},
		{"linkedin slug on subdomain", "https://uk.linkedin.com/jobs/view/senior-go-engineer-at-acme-3912345678", "linkedin", li, true},
		{"linkedin currentJobId", "https://www.linkedin.com/jobs/search/?keywords=go&currentJobId=3912345678", "linkedin", li, true},
		{"linkedin collections currentJobId", "https://www.linkedin.com/jobs/collections/recommended/?currentJobId=3912345678", "linkedin", li, true},
		{"linkedin search without id", "https://www.linkedin.com/jobs/search/?keywords=go", "", "", false},
		{"linkedin non-numeric id", "https://www.linkedin.com/jobs/view/abc", "", "", false},
		{"linkedin non-numeric currentJobId", "https://www.linkedin.com/jobs/search/?currentJobId=abc", "", "", false},
		{"lookalike host", "https://notlinkedin.com/jobs/view/3912345678", "", "", false},
		{"wis details", "https://workinstartups.com/details/5595385926", "wis", "https://workinstartups.com/details/5595385926", true},
		{"wis drops query and fragment", "https://workinstartups.com/details/5595385926?ref=x#apply", "wis", "https://workinstartups.com/details/5595385926", true},
		{"wis non-numeric id", "https://workinstartups.com/details/abc", "", "", false},
		{"wis other path", "https://workinstartups.com/jobs", "", "", false},
		{"hiring.cafe", "https://hiring.cafe/viewjob/abc123", "", "", false},
		{"indeed", "https://uk.indeed.com/viewjob?jk=abc123", "", "", false},
		{"ats host", "https://boards.greenhouse.io/acme/jobs/1001", "", "", false},
		{"not a url", "://bad", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source, canon, ok := detect.ResolveListing(tt.url)
			if ok != tt.ok || source != tt.source || canon != tt.canon {
				t.Errorf("ResolveListing(%q) = %q, %q, %v; want %q, %q, %v",
					tt.url, source, canon, ok, tt.source, tt.canon, tt.ok)
			}
		})
	}
}
