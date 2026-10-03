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
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

const (
	defaultPageLimit = 50
	defaultSinceDays = 90
	maxSinceDays     = 36500
	maxSeenJobIDs    = 5000
)

type jobCursor struct {
	Time time.Time `json:"time"`
	ID   string    `json:"id"`
}

type Service struct {
	store Store
	queue QueuePublisher
}

func NewService(store Store, q QueuePublisher) *Service {
	return &Service{store: store, queue: q}
}

func (s *Service) List(ctx context.Context, userID string, q dto.JobsQuery) (dto.JobPage, error) {
	limit, err := parsePageLimit(q.Limit)
	if err != nil {
		return dto.JobPage{}, err
	}
	if q.Availability != "" && q.Availability != "open" && q.Availability != "closed" && q.Availability != "all" {
		return dto.JobPage{}, apperr.Invalid("invalid availability")
	}

	sinceDays := int32(defaultSinceDays)
	if q.SinceDays != "" {
		d, err := strconv.ParseInt(q.SinceDays, 10, 32)
		if err != nil || d < 0 || d > maxSinceDays {
			return dto.JobPage{}, apperr.Invalid("since_days must be between 0 and 36500")
		}
		sinceDays = int32(d)
	}

	options := dto.JobPageOptions{Limit: int32(limit + 1), Availability: q.Availability, CompanyID: q.CompanyID, ScoredOnly: q.Scored == "1", SinceDays: sinceDays}
	if q.Cursor != "" {
		var decoded jobCursor
		if err := decodeCursor(q.Cursor, &decoded); err != nil || decoded.Time.IsZero() || decoded.ID == "" {
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
		page.NextCursor = encodeCursor(jobCursor{Time: last.ScrapedAt, ID: last.ID})
	}
	if page.Items == nil {
		page.Items = []dto.Job{}
	}
	return page, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (dto.Job, error) {
	job, err := s.store.GetJob(ctx, id, userID)
	switch {
	case errors.Is(err, data.ErrNotFound):
		return dto.Job{}, apperr.NotFound("job not found")
	case errors.Is(err, store.ErrInvalidID):
		return dto.Job{}, apperr.Invalid("invalid job ID")
	default:
		return job, err
	}
}

func (s *Service) MarkSeen(ctx context.Context, userID string, in dto.SeenInput) (struct{}, error) {
	if len(in.JobIDs) > maxSeenJobIDs {
		return struct{}{}, apperr.Invalid("too many job IDs")
	}
	err := s.store.MarkJobsSeen(ctx, userID, in.JobIDs, in.Seen)
	if errors.Is(err, store.ErrInvalidID) {
		return struct{}{}, apperr.Invalid("invalid job ID")
	}
	return struct{}{}, err
}

func parsePageLimit(raw string) (int, error) {
	if raw == "" {
		return defaultPageLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > 100 {
		return 0, apperr.Invalid("limit must be between 1 and 100")
	}
	return limit, nil
}

func decodeCursor(raw string, into any) error {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, into)
}

func encodeCursor(c any) string {
	data, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(data)
}
