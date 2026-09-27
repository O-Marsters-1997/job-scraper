package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
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
	j.CompanyID = uuidString(row.CompanyID)
	j.BoardID = uuidString(row.PrimaryBoardID)
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j
}

func fromListRow(row pgsqlc.ListJobsRow) (dto.Job, error) {
	j := dto.Job{
		ID:              row.ID.String(),
		Title:           row.Title,
		Location:        row.Location,
		URL:             row.Url,
		CompanySlug:     row.CompanySlug,
		Source:          row.Source,
		UpdatedAt:       row.UpdatedAt.Time,
		ScrapedAt:       row.ScrapedAt.Time,
		SalaryRaw:       row.SalaryRaw,
		WorkArrangement: row.WorkArrangement,
	}
	if err := unmarshalBreakdown(row.Breakdown, &j.Breakdown); err != nil {
		return dto.Job{}, err
	}
	j.SuitabilityScore = optionalInt32(row.SuitabilityScore)
	j.CompanyID = uuidString(row.CompanyID)
	j.BoardID = uuidString(row.PrimaryBoardID)
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j, nil
}

func optionalInt32(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

func unmarshalBreakdown(raw []byte, out *[]dto.ScoreRow) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("unmarshal job score breakdown: %w", err)
	}
	return nil
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
		job, err := fromListRow(row)
		if err != nil {
			return nil, fmt.Errorf("db.List: %w", err)
		}
		jobs[i] = job
	}
	return jobs, nil
}

func (db *DB) Page(ctx context.Context, userID string, options providers.JobPageOptions) (providers.JobPage, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return providers.JobPage{}, providers.ErrInvalidID
	}
	params := pgsqlc.PageJobsParams{UserID: uid, Availability: options.Availability, PageLimit: options.Limit}
	if params.Availability == "" {
		params.Availability = "open"
	}
	if options.CursorID != "" {
		params.CursorID, err = parseUUID(options.CursorID)
		if err != nil {
			return providers.JobPage{}, providers.ErrInvalidID
		}
		params.CursorTime = pgtype.Timestamptz{Time: options.CursorTime, Valid: true}
	}
	if options.CompanyID != "" {
		params.CompanyID, err = parseUUID(options.CompanyID)
		if err != nil {
			return providers.JobPage{}, providers.ErrInvalidID
		}
	}
	rows, err := db.queries.PageJobs(ctx, params)
	if err != nil {
		return providers.JobPage{}, fmt.Errorf("db.Page: %w", err)
	}
	page := providers.JobPage{Items: make([]dto.Job, len(rows))}
	for i, row := range rows {
		job, err := fromPageRow(row)
		if err != nil {
			return providers.JobPage{}, fmt.Errorf("db.Page: %w", err)
		}
		page.Items[i] = job
	}
	return page, nil
}

func fromPageRow(row pgsqlc.PageJobsRow) (dto.Job, error) {
	j := dto.Job{
		ID:              row.ID.String(),
		Title:           row.Title,
		Location:        row.Location,
		URL:             row.Url,
		CompanySlug:     row.CompanySlug,
		Source:          row.Source,
		UpdatedAt:       row.UpdatedAt.Time,
		ScrapedAt:       row.ScrapedAt.Time,
		SalaryRaw:       row.SalaryRaw,
		WorkArrangement: row.WorkArrangement,
	}
	if err := unmarshalBreakdown(row.Breakdown, &j.Breakdown); err != nil {
		return dto.Job{}, err
	}
	j.SuitabilityScore = optionalInt32(row.SuitabilityScore)
	j.CompanyID = uuidString(row.CompanyID)
	j.BoardID = uuidString(row.PrimaryBoardID)
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j, nil
}

func fromGetJobRow(row pgsqlc.GetJobRow) (dto.Job, error) {
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
	if err := unmarshalBreakdown(row.Breakdown, &j.Breakdown); err != nil {
		return dto.Job{}, err
	}
	j.SuitabilityScore = optionalInt32(row.SuitabilityScore)
	j.CompanyID = uuidString(row.CompanyID)
	j.BoardID = uuidString(row.PrimaryBoardID)
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j, nil
}

func (db *DB) GetJob(ctx context.Context, jobID, userID string) (dto.Job, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return dto.Job{}, providers.ErrInvalidID
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.Job{}, providers.ErrInvalidID
	}
	row, err := db.queries.GetJob(ctx, pgsqlc.GetJobParams{ID: jid, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Job{}, data.ErrNotFound
	}
	if err != nil {
		return dto.Job{}, fmt.Errorf("db.GetJob: %w", err)
	}
	job, err := fromGetJobRow(row)
	if err != nil {
		return dto.Job{}, fmt.Errorf("db.GetJob: %w", err)
	}
	return job, nil
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
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin mark jobs closed: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	closed, err := db.queries.WithTx(tx).MarkJobsClosed(ctx, urls)
	if err != nil {
		return fmt.Errorf("db.MarkJobsClosed: %w", err)
	}
	if len(closed) > 0 {
		ids := make([]string, len(closed))
		for i, id := range closed {
			ids[i] = id.String()
		}
		if err := db.scoring.JobsClosed(ctx, tx, ids); err != nil {
			return fmt.Errorf("scoring.JobsClosed: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit mark jobs closed: %w", err)
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
