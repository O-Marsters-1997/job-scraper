package jobsearch

import (
	"context"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	overviewTopLimit  = 5
	overviewTopWindow = 7 * 24 * time.Hour
)

func (s *Service) Overview(ctx context.Context, userID string, q dto.JobOverviewQuery) (dto.JobOverview, error) {
	now := time.Now()
	dayStart, err := parseDayStart(q.DayStart, now)
	if err != nil {
		return dto.JobOverview{}, err
	}
	ranked, err := oneOf(q.Ranked, "ranked", "1", "1", "0")
	if err != nil {
		return dto.JobOverview{}, err
	}
	excluded, err := s.excludedCompanySlugs(ctx, userID)
	if err != nil {
		return dto.JobOverview{}, err
	}

	stats, err := s.store.JobStats(ctx, userID, dto.JobStatsOptions{Now: now, DayStart: dayStart, ExcludedCompanySlugs: excluded})
	if err != nil {
		return dto.JobOverview{}, err
	}

	top := dto.JobPageOptions{
		Limit:                overviewTopLimit,
		Now:                  now,
		Sort:                 dto.JobSortNewest,
		ScoredOnly:           true,
		Seen:                 "unseen",
		ExcludedCompanySlugs: excluded,
	}
	if ranked == "1" {
		since := now.Add(-overviewTopWindow)
		top.Sort = dto.JobSortBest
		top.Bands = []string{"great", "good"}
		top.ScrapedSince = &since
	}
	page, err := s.store.Page(ctx, userID, top)
	if err != nil {
		return dto.JobOverview{}, err
	}
	if page.Items == nil {
		page.Items = []dto.Job{}
	}
	return dto.JobOverview{JobStats: stats, Top: page.Items}, nil
}

func parseDayStart(raw string, now time.Time) (time.Time, error) {
	if raw == "" {
		return now.UTC().Truncate(24 * time.Hour), nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, apperr.Invalid("day_start must be an RFC 3339 timestamp")
	}
	return t, nil
}
