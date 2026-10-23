package jobsearch

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

func (s *Service) ListScrapeRuns(ctx context.Context, userID string, q dto.ScrapeRunsQuery) (dto.ScrapeRunPage, error) {
	limit, err := parsePageLimit(q.Limit)
	if err != nil {
		return dto.ScrapeRunPage{}, err
	}
	filter, _, err := parseScrapeRunFilter(q, time.Now())
	if err != nil {
		return dto.ScrapeRunPage{}, err
	}
	opts := dto.ScrapeRunListOptions{ScrapeRunFilter: filter, Limit: limit + 1}
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

func (s *Service) ScrapeRunSummary(ctx context.Context, userID string, q dto.ScrapeRunsQuery) (dto.ScrapeRunSummary, error) {
	filter, end, err := parseScrapeRunFilter(q, time.Now())
	if err != nil {
		return dto.ScrapeRunSummary{}, err
	}
	span := end.Sub(filter.From)
	bucket := scrapeRunBuckets[len(scrapeRunBuckets)-1]
	for _, b := range scrapeRunBuckets {
		if span <= maxHistogramBuckets*b {
			bucket = b
			break
		}
	}
	summary, err := s.store.ScrapeRunSummary(ctx, userID, dto.ScrapeRunSummaryOptions{ScrapeRunFilter: filter, Bucket: bucket, End: end})
	if errors.Is(err, store.ErrInvalidID) {
		return dto.ScrapeRunSummary{}, apperr.Invalid("invalid target")
	}
	return summary, err
}

const (
	maxHistogramBuckets = 48
	maxCustomRange      = 30 * 24 * time.Hour
)

var scrapeRunBuckets = []time.Duration{
	5 * time.Minute, 15 * time.Minute, time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
}

var scrapeRunRanges = map[string]time.Duration{
	"1h": time.Hour, "6h": 6 * time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour, "30d": maxCustomRange,
}

var scrapeRunOutcomes = []string{
	string(dto.OutcomeRunning), string(dto.OutcomeSucceeded), string(dto.OutcomeRetried), string(dto.OutcomeFailed),
}

var scrapeRunTriggers = []string{string(dto.TriggerScheduled), string(dto.TriggerManual)}

func parseScrapeRunFilter(q dto.ScrapeRunsQuery, now time.Time) (dto.ScrapeRunFilter, time.Time, error) {
	filter := dto.ScrapeRunFilter{TargetID: q.Target, Q: strings.TrimSpace(q.Q)}
	var err error
	for _, facet := range []struct {
		name, raw string
		valid     func(string) bool
		dst       *[]string
	}{
		{"outcome", q.Outcome, inSet(scrapeRunOutcomes), &filter.Outcome},
		{"kind", q.Kind, inSet([]string{string(dto.ScrapeRunSearch)}), &filter.Kind},
		{"source", q.Source, func(v string) bool { _, ok := sourcespec.SourceRole(v); return ok }, &filter.Source},
		{"trigger", q.Trigger, inSet(scrapeRunTriggers), &filter.Trigger},
	} {
		if *facet.dst, err = parseFacet(facet.name, facet.raw, facet.valid); err != nil {
			return dto.ScrapeRunFilter{}, time.Time{}, err
		}
	}
	if q.From != "" || q.To != "" {
		from, errFrom := time.Parse(time.RFC3339, q.From)
		to, errTo := time.Parse(time.RFC3339, q.To)
		if errFrom != nil || errTo != nil || !to.After(from) || to.Sub(from) > maxCustomRange {
			return dto.ScrapeRunFilter{}, time.Time{}, apperr.Invalid("from and to must be RFC 3339 times at most 30 days apart")
		}
		filter.From, filter.To = from, to
		return filter, to, nil
	}
	if q.Range == "" {
		q.Range = "7d"
	}
	d, ok := scrapeRunRanges[q.Range]
	if !ok {
		return dto.ScrapeRunFilter{}, time.Time{}, apperr.Invalid("range must be 1h, 6h, 24h, 7d or 30d")
	}
	filter.From = now.Add(-d)
	return filter, now, nil
}

func inSet(allowed []string) func(string) bool {
	return func(v string) bool { return slices.Contains(allowed, v) }
}

func parseFacet(name, raw string, valid func(string) bool) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	values := strings.Split(raw, ",")
	for _, v := range values {
		if !valid(v) {
			return nil, apperr.Invalid("unknown " + name + " " + strconv.Quote(v))
		}
	}
	return values, nil
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
