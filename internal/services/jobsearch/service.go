// Package jobsearch is the jobsearch context: jobs, companies, company
// boards, source targets, candidates and ingest (ADR 0011).
package jobsearch

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

const defaultJobPageLimit = 50

type jobCursor struct {
	Time time.Time `json:"time"`
	ID   string    `json:"id"`
}

type jobStore interface {
	Page(ctx context.Context, userID string, options dto.JobPageOptions) (dto.JobPage, error)
	GetJob(ctx context.Context, jobID, userID string) (dto.Job, error)
}

// Service is the jobs feature: the score-sorted, cursor-paginated job page
// and single-job reads.
type Service struct {
	store jobStore
}

func NewService(store jobStore) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context, userID string, q dto.JobsQuery) (dto.JobPage, error) {
	limit := defaultJobPageLimit
	if q.Limit != "" {
		l, err := strconv.Atoi(q.Limit)
		if err != nil || l < 1 || l > 100 {
			return dto.JobPage{}, apperr.Invalid("limit must be between 1 and 100")
		}
		limit = l
	}
	if q.Availability != "" && q.Availability != "open" && q.Availability != "closed" && q.Availability != "all" {
		return dto.JobPage{}, apperr.Invalid("invalid availability")
	}

	options := dto.JobPageOptions{Limit: int32(limit + 1), Availability: q.Availability, CompanyID: q.CompanyID}
	if q.Cursor != "" {
		decoded, err := decodeJobCursor(q.Cursor)
		if err != nil {
			return dto.JobPage{}, apperr.Invalid("invalid cursor")
		}
		options.CursorTime, options.CursorID = decoded.Time, decoded.ID
	}

	page, err := s.store.Page(ctx, userID, options)
	if err != nil {
		if errors.Is(err, store.ErrInvalidID) {
			return dto.JobPage{}, apperr.Invalid("invalid company or cursor ID")
		}
		return dto.JobPage{}, err
	}

	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeJobCursor(jobCursor{Time: last.ScrapedAt, ID: last.ID})
	}
	if page.Items == nil {
		page.Items = []dto.Job{}
	}
	return page, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (dto.Job, error) {
	job, err := s.store.GetJob(ctx, id, userID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return dto.Job{}, apperr.NotFound("job not found")
	case errors.Is(err, store.ErrInvalidID):
		return dto.Job{}, apperr.Invalid("invalid job ID")
	default:
		return job, err
	}
}

func decodeJobCursor(raw string) (jobCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return jobCursor{}, err
	}
	var c jobCursor
	if err := json.Unmarshal(data, &c); err != nil {
		return jobCursor{}, err
	}
	if c.Time.IsZero() || c.ID == "" {
		return jobCursor{}, errors.New("empty cursor")
	}
	return c, nil
}

func encodeJobCursor(c jobCursor) string {
	data, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(data)
}
