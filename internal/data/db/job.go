package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func toUpsertParams(j dto.Job) pgsqlc.UpsertJobParams {
	return pgsqlc.UpsertJobParams{
		Title:           j.Title,
		Location:        j.Location,
		Url:             j.URL,
		CompanySlug:     j.CompanySlug,
		Source:          j.Source,
		UpdatedAt:       pgtype.Timestamptz{Time: j.UpdatedAt, Valid: true},
		Description:     j.Description,
		SalaryRaw:       j.SalaryRaw,
		WorkArrangement: j.WorkArrangement,
	}
}

func fromRow(row pgsqlc.Job) dto.Job {
	j := dto.Job{
		ID:              row.ID.String(),
		Title:           row.Title,
		Location:        row.Location,
		URL:             row.Url,
		CompanySlug:     row.CompanySlug,
		Source:          row.Source,
		UpdatedAt:       row.UpdatedAt.Time,
		ScrapedAt:       row.ScrapedAt.Time,
		Description:     row.Description,
		SalaryRaw:       row.SalaryRaw,
		WorkArrangement: row.WorkArrangement,
	}
	if row.CompanyID.Valid {
		j.CompanyID = row.CompanyID.String()
	}
	if row.PrimaryBoardID.Valid {
		j.BoardID = row.PrimaryBoardID.String()
	}
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j
}

func fromListRow(row pgsqlc.ListJobsRow) dto.Job {
	j := dto.Job{
		ID:                 row.ID.String(),
		Title:              row.Title,
		Location:           row.Location,
		URL:                row.Url,
		CompanySlug:        row.CompanySlug,
		Source:             row.Source,
		UpdatedAt:          row.UpdatedAt.Time,
		ScrapedAt:          row.ScrapedAt.Time,
		Description:        row.Description,
		SalaryRaw:          row.SalaryRaw,
		WorkArrangement:    row.WorkArrangement,
		Matched:            row.Matched,
		Missing:            row.Missing,
		SuitabilitySkipped: row.SuitabilitySkipped,
	}
	if row.RelevanceScore.Valid {
		v := int(row.RelevanceScore.Int32)
		j.RelevanceScore = &v
	}
	if row.SuitabilityScore.Valid {
		v := int(row.SuitabilityScore.Int32)
		j.SuitabilityScore = &v
	}
	if row.Reasoning.Valid {
		j.Reasoning = &row.Reasoning.String
	}
	if row.CompanyID.Valid {
		j.CompanyID = row.CompanyID.String()
	}
	if row.PrimaryBoardID.Valid {
		j.BoardID = row.PrimaryBoardID.String()
	}
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j
}

func (db *DB) Save(ctx context.Context, jobs []dto.Job) ([]dto.Job, error) {
	if len(jobs) == 0 {
		return nil, nil
	}
	out := make([]dto.Job, 0, len(jobs))
	for _, j := range jobs {
		row, err := db.queries.UpsertJob(ctx, toUpsertParams(j))
		if err != nil {
			return nil, fmt.Errorf("db.Save: %w", err)
		}
		j.ID = row.ID.String()
		out = append(out, j)
	}
	slog.Debug("jobs saved", slog.Int("count", len(out)))
	return out, nil
}

func (db *DB) NewURLs(ctx context.Context, urls []string) ([]string, error) {
	existing, err := db.queries.ExistingURLs(ctx, urls)
	if err != nil {
		return nil, fmt.Errorf("db.NewURLs: %w", err)
	}
	known := make(map[string]struct{}, len(existing))
	for _, u := range existing {
		known[u] = struct{}{}
	}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if _, ok := known[u]; !ok {
			out = append(out, u)
		}
	}
	return out, nil
}

func (db *DB) List(ctx context.Context, userID string) ([]dto.Job, error) {
	var uid pgtype.UUID
	if userID != "" {
		var err error
		uid, err = parseUUID(userID)
		if err != nil {
			return nil, fmt.Errorf("db.List: %w", err)
		}
	}
	rows, err := db.queries.ListJobs(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.List: %w", err)
	}
	jobs := make([]dto.Job, len(rows))
	for i, row := range rows {
		jobs[i] = fromListRow(row)
	}
	return jobs, nil
}

func fromGetJobRow(row pgsqlc.GetJobRow) dto.Job {
	j := dto.Job{
		ID:                 row.ID.String(),
		Title:              row.Title,
		Location:           row.Location,
		URL:                row.Url,
		CompanySlug:        row.CompanySlug,
		Source:             row.Source,
		UpdatedAt:          row.UpdatedAt.Time,
		ScrapedAt:          row.ScrapedAt.Time,
		Description:        row.Description,
		SalaryRaw:          row.SalaryRaw,
		WorkArrangement:    row.WorkArrangement,
		Matched:            row.Matched,
		Missing:            row.Missing,
		SuitabilitySkipped: row.SuitabilitySkipped,
	}
	if row.RelevanceScore.Valid {
		v := int(row.RelevanceScore.Int32)
		j.RelevanceScore = &v
	}
	if row.SuitabilityScore.Valid {
		v := int(row.SuitabilityScore.Int32)
		j.SuitabilityScore = &v
	}
	if row.Reasoning.Valid {
		j.Reasoning = &row.Reasoning.String
	}
	if row.CompanyID.Valid {
		j.CompanyID = row.CompanyID.String()
	}
	if row.PrimaryBoardID.Valid {
		j.BoardID = row.PrimaryBoardID.String()
	}
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j
}

func (db *DB) GetJob(ctx context.Context, jobID, userID string) (dto.Job, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return dto.Job{}, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.Job{}, err
	}
	row, err := db.queries.GetJob(ctx, pgsqlc.GetJobParams{ID: jid, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Job{}, providers.ErrNotFound
	}
	if err != nil {
		return dto.Job{}, fmt.Errorf("db.GetJob: %w", err)
	}
	return fromGetJobRow(row), nil
}

func (db *DB) OpenJobURLsForBoard(ctx context.Context, source, companySlug string) ([]string, error) {
	urls, err := db.queries.OpenJobURLsForBoard(ctx, pgsqlc.OpenJobURLsForBoardParams{
		Source:      source,
		CompanySlug: companySlug,
	})
	if err != nil {
		return nil, fmt.Errorf("db.OpenJobURLsForBoard: %w", err)
	}
	return urls, nil
}

func (db *DB) MarkJobsClosed(ctx context.Context, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	if err := db.queries.MarkJobsClosed(ctx, urls); err != nil {
		return fmt.Errorf("db.MarkJobsClosed: %w", err)
	}
	return nil
}

func (db *DB) ListSince(ctx context.Context, since time.Time) ([]dto.Job, error) {
	rows, err := db.queries.ListJobsSince(ctx, pgtype.Timestamptz{Time: since, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("db.ListSince: %w", err)
	}
	jobs := make([]dto.Job, len(rows))
	for i, row := range rows {
		jobs[i] = fromRow(row)
	}
	return jobs, nil
}
