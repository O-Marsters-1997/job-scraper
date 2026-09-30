package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// BoardSpec describes one board integration. URL is the API endpoint; Parse decodes
// the response body into jobs. FetchPage stamps Source with Name, and CompanySlug when set.
type BoardSpec struct {
	Name     string
	UseProxy bool
	// Post, when true, fetches the board via POST with an empty JSON body
	// instead of GET. Workable's job-list API only responds to POST.
	Post        bool
	URL         string
	CompanySlug string
	Parse       func(body []byte) ([]dto.Job, error)
}

// BoardSource is the shared implementation for ATS "board" sources whose only
// per-source variation is the endpoint URL, the decoder, and the field mapping — all
// captured by BoardSpec.
type BoardSource struct {
	PaginatedBase
	spec BoardSpec
}

var _ Source = (*BoardSource)(nil)

func NewBoardSource(spec BoardSpec) *BoardSource {
	return &BoardSource{
		PaginatedBase: NewBase(Config{Name: spec.Name, UseProxy: spec.UseProxy}),
		spec:          spec,
	}
}

func (b *BoardSource) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	if cursor != "" {
		return nil, "", fmt.Errorf("%s: unexpected cursor %q", b.spec.Name, cursor)
	}
	fetch := b.Get
	if b.spec.Post {
		fetch = b.PostEmptyJSON
	}
	body, err := fetch(ctx, b.spec.URL)
	if err != nil {
		return nil, "", err
	}
	jobs, err := b.spec.Parse(body)
	for i := range jobs {
		jobs[i].Source = b.spec.Name
		if b.spec.CompanySlug != "" {
			jobs[i].CompanySlug = b.spec.CompanySlug
		}
	}
	return jobs, "", err
}

// RFC3339OrNow parses an RFC3339 timestamp, falling back to the current UTC time when
// raw is empty or unparseable.
func RFC3339OrNow(raw string) time.Time {
	if raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}
