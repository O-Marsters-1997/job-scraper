package sources

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	boards []string
	spec   BoardSpec
}

var _ Source = (*BoardSource)(nil)

func NewBoardSource(boards []string, spec BoardSpec) *BoardSource {
	return &BoardSource{
		PaginatedBase: NewBase(Config{Name: spec.Name, UseProxy: spec.UseProxy}),
		boards:        boards,
		spec:          spec,
	}
}

func (b *BoardSource) FetchPage(ctx context.Context, cursor string) (Page, error) {
	if len(b.boards) != 1 {
		return Page{}, fmt.Errorf("%s: expected one board, got %d", b.spec.Name, len(b.boards))
	}
	if cursor != "" {
		return Page{}, fmt.Errorf("%s: unexpected cursor %q", b.spec.Name, cursor)
	}
	jobs, err := b.fetchBoard(ctx, b.boards[0])
	return Page{Jobs: jobs}, err
}

func (b *BoardSource) fetchBoard(ctx context.Context, token string) ([]dto.Job, error) {
	fetch := b.Get
	if b.spec.Post {
		fetch = b.PostEmptyJSON
	}
	body, err := fetch(ctx, b.spec.URL(token))
	if err != nil {
		return nil, err
	}
	return b.spec.Parse(body, token)
}

// Iterate fetches every configured board token, logging and continuing past
// per-token failures so one dead board doesn't starve the rest of the ATS's
// boards. Errors are joined and returned once all tokens have been attempted.
func (b *BoardSource) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	var errs []error
	for _, token := range b.boards {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		jobs, err := b.fetchBoard(ctx, token)
		if err != nil {
			slog.Warn("board parse failed", slog.String("source", b.spec.Name), slog.String("board", token), slog.Any("err", err))
			errs = append(errs, fmt.Errorf("%s: board %s: %w", b.spec.Name, token, err))
			continue
		}
		stop, err := fn(ctx, jobs)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: board %s: %w", b.spec.Name, token, err))
			continue
		}
		if stop {
			break
		}
	}
	return errors.Join(errs...)
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
