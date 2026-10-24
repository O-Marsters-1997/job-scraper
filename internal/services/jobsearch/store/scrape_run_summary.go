package store

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
)

type runFilter struct {
	target   pgtype.UUID
	from, to pgtype.Timestamptz
	q        pgtype.Text
}

func toRunFilter(f dto.ScrapeRunFilter) (runFilter, error) {
	var out runFilter
	if f.TargetID != "" {
		id, err := data.UUID(f.TargetID)
		if err != nil {
			return out, ErrInvalidID
		}
		out.target = id
	}
	if !f.From.IsZero() {
		out.from = pgtype.Timestamptz{Time: f.From, Valid: true}
	}
	if !f.To.IsZero() {
		out.to = pgtype.Timestamptz{Time: f.To, Valid: true}
	}
	if f.Q != "" {
		out.q = pgtype.Text{String: likeEscaper.Replace(f.Q), Valid: true}
	}
	return out, nil
}

func (s *Store) ScrapeRunSummary(ctx context.Context, userID string, opts dto.ScrapeRunSummaryOptions) (dto.ScrapeRunSummary, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.ScrapeRunSummary{}, ErrInvalidID
	}
	if opts.Bucket <= 0 || opts.From.IsZero() {
		return dto.ScrapeRunSummary{}, fmt.Errorf("store.ScrapeRunSummary: bucket %v and from %v must be set", opts.Bucket, opts.From)
	}
	f, err := toRunFilter(opts.ScrapeRunFilter)
	if err != nil {
		return dto.ScrapeRunSummary{}, err
	}
	rows, err := s.queries.SummariseScrapeRuns(ctx, sqlc.SummariseScrapeRunsParams{
		BucketSecs: opts.Bucket.Seconds(), UserID: uid, TargetID: f.target, FromAt: f.from, ToAt: f.to, Q: f.q,
	})
	if err != nil {
		return dto.ScrapeRunSummary{}, fmt.Errorf("store.ScrapeRunSummary: %w", err)
	}
	return summariseRuns(rows, opts), nil
}

func summariseRuns(rows []sqlc.SummariseScrapeRunsRow, opts dto.ScrapeRunSummaryOptions) dto.ScrapeRunSummary {
	selected := map[dto.ScrapeRunFacet][]string{
		dto.FacetOutcome: opts.Outcome, dto.FacetKind: opts.Kind,
		dto.FacetSource: opts.Source, dto.FacetTrigger: opts.Trigger,
	}
	summary := dto.ScrapeRunSummary{
		Facets: map[dto.ScrapeRunFacet]map[string]int{
			dto.FacetOutcome: {}, dto.FacetKind: {}, dto.FacetSource: {}, dto.FacetTrigger: {},
		},
		BucketMs: opts.Bucket.Milliseconds(),
	}
	start := opts.From.Truncate(opts.Bucket)
	for t := start; t.Before(opts.End); t = t.Add(opts.Bucket) {
		summary.Histogram = append(summary.Histogram, dto.HistogramBucket{Start: t})
	}

	for _, row := range rows {
		values := map[dto.ScrapeRunFacet]string{
			dto.FacetOutcome: row.Outcome, dto.FacetKind: row.Kind,
			dto.FacetSource: row.Source, dto.FacetTrigger: row.Trigger,
		}
		var failing []dto.ScrapeRunFacet
		for facet, value := range values {
			if sel := selected[facet]; len(sel) > 0 && !slices.Contains(sel, value) {
				failing = append(failing, facet)
			}
		}
		n := int(row.Runs)
		switch len(failing) {
		case 0:
			summary.Total += n
			for facet, value := range values {
				summary.Facets[facet][value] += n
			}
			if i := int(row.Bucket.Time.Sub(start) / opts.Bucket); i >= 0 && i < len(summary.Histogram) {
				b := &summary.Histogram[i]
				switch row.Outcome {
				case string(dto.OutcomeFailed):
					b.Failed += n
				case string(dto.OutcomeRetried):
					b.Retried += n
				default:
					b.OK += n
				}
			}
		case 1:
			summary.Facets[failing[0]][values[failing[0]]] += n
		}
	}
	return summary
}
