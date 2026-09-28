package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// BoardSpec describes one ATS board integration. URL builds the API endpoint for a
// board token; Parse decodes the response body into jobs.
type BoardSpec struct {
	Name     string
	UseProxy bool
	// Post, when true, fetches the board via POST with an empty JSON body
	// instead of GET. Workable's job-list API only responds to POST.
	Post  bool
	URL   func(token string) string
	Parse func(body []byte, token string) ([]dto.Job, error)
}

// BoardSource is the shared implementation for ATS "board" sources whose only
// per-source variation is the endpoint URL, the decoder, and the field mapping — all
// captured by BoardSpec.
type BoardSource struct {
	PaginatedBase
	board string
	spec  BoardSpec
}

var _ Source = (*BoardSource)(nil)

func NewBoardSource(board string, spec BoardSpec) *BoardSource {
	return &BoardSource{
		PaginatedBase: NewBase(Config{Name: spec.Name, UseProxy: spec.UseProxy}),
		board:         board,
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
	body, err := fetch(ctx, b.spec.URL(b.board))
	if err != nil {
		return nil, "", err
	}
	jobs, err := b.spec.Parse(body, b.board)
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
