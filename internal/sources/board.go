package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// BoardSpec describes one ATS board integration. URL builds the API endpoint for a
// board token; Parse decodes the response body into jobs. Schedule and
// MinScrapeInterval come from NewBase defaults; set UseProxy only when the board needs
// the Web Unlocker.
type BoardSpec struct {
	Name      string
	URLPrefix string
	UseProxy  bool
	URL       func(token string) string
	Parse     func(body []byte, token string) ([]dto.Job, error)
}

// BoardSource is the shared implementation for ATS "board" sources whose only
// per-source variation is the endpoint URL, the decoder, and the field mapping — all
// captured by BoardSpec.
type BoardSource struct {
	PaginatedBase
	boards []string
	spec   BoardSpec
}

var _ Source = (*BoardSource)(nil)

func NewBoardSource(boards []string, spec BoardSpec) *BoardSource {
	return &BoardSource{
		PaginatedBase: NewBase(Config{Name: spec.Name, URLPrefix: spec.URLPrefix, UseProxy: spec.UseProxy}),
		boards:        boards,
		spec:          spec,
	}
}

func (b *BoardSource) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	for _, token := range b.boards {
		body, err := b.Get(ctx, b.spec.URL(token))
		if err != nil {
			return fmt.Errorf("%s: board %s: %w", b.spec.Name, token, err)
		}
		jobs, err := b.spec.Parse(body, token)
		if err != nil {
			return fmt.Errorf("%s: board %s: %w", b.spec.Name, token, err)
		}
		if stop, err := fn(ctx, jobs); err != nil || stop {
			return err
		}
	}
	return nil
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
