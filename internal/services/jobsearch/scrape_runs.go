package jobsearch

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

func (s *Service) ListScrapeRuns(ctx context.Context, userID string, q dto.ScrapeRunsQuery) (dto.ScrapeRunPage, error) {
	limit, err := parsePageLimit(q.Limit)
	if err != nil {
		return dto.ScrapeRunPage{}, err
	}
	opts := dto.ScrapeRunListOptions{TargetID: q.Target, Limit: limit + 1}
	if q.Before != "" {
		if opts.BeforeStartedAt, opts.BeforeID, err = decodeScrapeRunCursor(q.Before); err != nil {
			return dto.ScrapeRunPage{}, err
		}
	}
	runs, err := s.store.ListScrapeRuns(ctx, userID, opts)
	if errors.Is(err, store.ErrInvalidID) {
		return dto.ScrapeRunPage{}, apperr.Invalid("invalid target or cursor")
	}
	if err != nil {
		return dto.ScrapeRunPage{}, err
	}
	page := dto.ScrapeRunPage{Items: runs}
	if len(runs) > limit {
		page.Items = runs[:limit]
		last := page.Items[limit-1]
		cursor := encodeScrapeRunCursor(last)
		page.NextCursor = &cursor
	}
	if page.Items == nil {
		page.Items = []dto.ScrapeRun{}
	}
	return page, nil
}

func encodeScrapeRunCursor(run dto.ScrapeRun) string {
	return base64.RawURLEncoding.EncodeToString([]byte(run.StartedAt.Format(time.RFC3339Nano) + "|" + run.ID))
}

func decodeScrapeRunCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	startedAt, id, ok := strings.Cut(string(raw), "|")
	if err != nil || !ok || id == "" {
		return time.Time{}, "", apperr.Invalid("invalid cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return time.Time{}, "", apperr.Invalid("invalid cursor")
	}
	return t, id, nil
}

func (m *Module) ScrapeRunPageDone(ctx context.Context, runID string) error {
	return m.store.ScrapeRunPageDone(ctx, runID)
}

func (m *Module) ScrapeRunDeferred(ctx context.Context, runID string) error {
	return m.store.ScrapeRunDeferred(ctx, runID)
}

func (m *Module) ScrapeRunAttemptFailed(ctx context.Context, runID string, page, attempt int, message string) error {
	return m.store.ScrapeRunAttemptFailed(ctx, runID, page, attempt, message)
}

func (m *Module) FinishScrapeRun(ctx context.Context, runID string, stop dto.StopReason) error {
	return m.store.FinishScrapeRun(ctx, runID, stop)
}

func (m *Module) DeleteExpiredScrapeRuns(ctx context.Context) error {
	return m.store.DeleteExpiredScrapeRuns(ctx)
}
