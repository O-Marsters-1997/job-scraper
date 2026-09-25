// Package jobs holds the domain rules for listing and reading jobs,
// including cursor-based pagination. Persistence goes through
// providers.JobProvider.
package jobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const defaultLimit = 50

type cursor struct {
	Time time.Time `json:"time"`
	ID   string    `json:"id"`
}

type Service struct {
	jobs providers.JobProvider
}

func New(jobs providers.JobProvider) *Service {
	return &Service{jobs: jobs}
}

// List returns a cursor-paginated page of the caller's jobs.
func (s *Service) List(ctx context.Context, userID string, q dto.JobsQuery) (providers.JobPage, error) {
	limit := defaultLimit
	if q.Limit != "" {
		l, err := strconv.Atoi(q.Limit)
		if err != nil || l < 1 || l > 100 {
			return providers.JobPage{}, apperr.Invalid("limit must be between 1 and 100")
		}
		limit = l
	}
	if q.Availability != "" && q.Availability != "open" && q.Availability != "closed" && q.Availability != "all" {
		return providers.JobPage{}, apperr.Invalid("invalid availability")
	}

	options := providers.JobPageOptions{Limit: int32(limit + 1), Availability: q.Availability, CompanyID: q.CompanyID}
	if q.Cursor != "" {
		decoded, err := decodeCursor(q.Cursor)
		if err != nil {
			return providers.JobPage{}, apperr.Invalid("invalid cursor")
		}
		options.CursorTime, options.CursorID = decoded.Time, decoded.ID
	}

	page, err := s.jobs.Page(ctx, userID, options)
	if err != nil {
		if errors.Is(err, providers.ErrInvalidID) {
			return providers.JobPage{}, apperr.Invalid("invalid company or cursor ID")
		}
		return providers.JobPage{}, err
	}

	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(cursor{Time: last.ScrapedAt, ID: last.ID})
	}
	if page.Items == nil {
		page.Items = []dto.Job{}
	}
	return page, nil
}

// Get returns a single job for the caller.
func (s *Service) Get(ctx context.Context, userID, id string) (dto.Job, error) {
	job, err := s.jobs.GetJob(ctx, id, userID)
	switch {
	case errors.Is(err, providers.ErrNotFound):
		return dto.Job{}, apperr.NotFound("job not found")
	case errors.Is(err, providers.ErrInvalidID):
		return dto.Job{}, apperr.Invalid("invalid job ID")
	default:
		return job, err
	}
}

func decodeCursor(raw string) (cursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return cursor{}, err
	}
	var c cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return cursor{}, err
	}
	if c.Time.IsZero() || c.ID == "" {
		return cursor{}, errors.New("empty cursor")
	}
	return c, nil
}

func encodeCursor(c cursor) string {
	data, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(data)
}
