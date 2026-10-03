package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// BoardSpec describes one board integration. URL is the API endpoint; Parse decodes
// the response body into jobs. PollBoard stamps Source with Name, and CompanySlug when set.
// Count, when set, reads how many jobs the ATS says the board holds.
type BoardSpec struct {
	Name  string
	Route Route
	// Post, when true, fetches the board via POST with an empty JSON body
	// instead of GET. Workable's job-list API only responds to POST.
	Post        bool
	URL         string
	CompanySlug string
	Header      http.Header
	Parse       func(body []byte) ([]dto.Job, error)
	Count       func(body []byte) (int, error)
}

// BoardSource is the shared implementation for ATS "board" sources whose only
// per-source variation is the endpoint URL, the decoder, and the field mapping — all
// captured by BoardSpec.
type BoardSource struct {
	PaginatedBase
	spec BoardSpec
}

var (
	_ Source      = (*BoardSource)(nil)
	_ BoardPoller = (*BoardSource)(nil)
)

func NewBoardSource(spec BoardSpec) *BoardSource {
	return &BoardSource{
		PaginatedBase: NewBase(Config{Name: spec.Name, Route: spec.Route, Header: spec.Header}),
		spec:          spec,
	}
}

func (b *BoardSource) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	if cursor != "" {
		return nil, "", fmt.Errorf("%s: unexpected cursor %q", b.spec.Name, cursor)
	}
	res, err := b.PollBoard(ctx)
	return res.Jobs, "", err
}

// PollBoard fetches the board once and returns its parsed jobs with the ATS-reported count.
// Reported is 0 when the spec has no Count.
func (b *BoardSource) PollBoard(ctx context.Context) (BoardResult, error) {
	fetch := b.Get
	if b.spec.Post {
		fetch = b.PostEmptyJSON
	}
	body, err := fetch(ctx, b.spec.URL)
	if err != nil {
		return BoardResult{}, err
	}
	jobs, err := b.spec.Parse(body)
	for i := range jobs {
		jobs[i].Source = b.spec.Name
		if b.spec.CompanySlug != "" {
			jobs[i].CompanySlug = b.spec.CompanySlug
		}
	}
	res := BoardResult{Jobs: jobs}
	if err != nil {
		return res, err
	}
	if b.spec.Count != nil {
		reported, err := b.spec.Count(body)
		if err != nil {
			return res, fmt.Errorf("%s: count: %w", b.spec.Name, err)
		}
		res.Reported = reported
	}
	return res, nil
}

// JSONArrayLen counts the elements of the array at key in a JSON object body, or of a
// top-level array when key is empty.
func JSONArrayLen(key string) func([]byte) (int, error) {
	return func(body []byte) (int, error) {
		raw := json.RawMessage(body)
		if key != "" {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(body, &obj); err != nil {
				return 0, fmt.Errorf("parse json: %w", err)
			}
			raw = obj[key]
		}
		var elems []json.RawMessage
		if err := json.Unmarshal(raw, &elems); err != nil {
			return 0, fmt.Errorf("parse json array: %w", err)
		}
		return len(elems), nil
	}
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
