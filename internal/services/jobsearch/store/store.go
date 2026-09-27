// Package store is the jobsearch context's Postgres store: jobs, companies,
// company boards, source targets and candidates, plus a score-sorted read-join
// onto job_scores for the job page.
package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/candidates"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
)

var (
	ErrNotFound           = apperr.NotFound("not found")
	ErrInvalidID          = errors.New("invalid ID")
	ErrCanonicalConflict  = errors.New("canonical job identity conflict")
	ErrBoardConflict      = apperr.Conflict("board belongs to another company")
	ErrSourceTargetExists = apperr.Conflict("source target already exists")
)

// ScoringWriter is scoring's tx-scoped facade, called from within
// jobsearch's own transactions instead of writing scoring's tables
// directly (ADR 0011).
type ScoringWriter interface {
	JobsChanged(ctx context.Context, tx pgx.Tx, jobIDs []string, firstDiscovery bool) error
	JobsClosed(ctx context.Context, tx pgx.Tx, jobIDs []string) error
	CompanyTracked(ctx context.Context, tx pgx.Tx, userID, companyID string) error
}

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	scoring ScoringWriter
}

func New(pool *pgxpool.Pool, scoring ScoringWriter) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool), scoring: scoring}
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}

func optionalUUID(id string) (pgtype.UUID, error) {
	if id == "" {
		return pgtype.UUID{}, nil
	}
	return parseUUID(id)
}

func optionalTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func (s *Store) ListJobs(ctx context.Context, userID string) ([]dto.Job, error) {
	var uid pgtype.UUID
	if userID != "" {
		var err error
		uid, err = parseUUID(userID)
		if err != nil {
			return nil, fmt.Errorf("store.ListJobs: %w", err)
		}
	}
	rows, err := s.queries.ListJobs(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListJobs: %w", err)
	}
	jobs := make([]dto.Job, len(rows))
	for i, row := range rows {
		job, err := toListJobDTO(row)
		if err != nil {
			return nil, fmt.Errorf("store.ListJobs: %w", err)
		}
		jobs[i] = job
	}
	return jobs, nil
}

func (s *Store) Page(ctx context.Context, userID string, options dto.JobPageOptions) (dto.JobPage, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.JobPage{}, ErrInvalidID
	}
	params := sqlc.PageJobsParams{UserID: uid, Availability: options.Availability, PageLimit: options.Limit}
	if params.Availability == "" {
		params.Availability = "open"
	}
	if options.CursorID != "" {
		params.CursorID, err = parseUUID(options.CursorID)
		if err != nil {
			return dto.JobPage{}, ErrInvalidID
		}
		params.CursorTime = pgtype.Timestamptz{Time: options.CursorTime, Valid: true}
	}
	if options.CompanyID != "" {
		params.CompanyID, err = parseUUID(options.CompanyID)
		if err != nil {
			return dto.JobPage{}, ErrInvalidID
		}
	}
	rows, err := s.queries.PageJobs(ctx, params)
	if err != nil {
		return dto.JobPage{}, fmt.Errorf("store.Page: %w", err)
	}
	page := dto.JobPage{Items: make([]dto.Job, len(rows))}
	for i, row := range rows {
		job, err := toPageJobDTO(row)
		if err != nil {
			return dto.JobPage{}, fmt.Errorf("store.Page: %w", err)
		}
		page.Items[i] = job
	}
	return page, nil
}

func (s *Store) GetJob(ctx context.Context, jobID, userID string) (dto.Job, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return dto.Job{}, ErrInvalidID
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.Job{}, ErrInvalidID
	}
	row, err := s.queries.GetJob(ctx, sqlc.GetJobParams{ID: jid, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Job{}, ErrNotFound
	}
	if err != nil {
		return dto.Job{}, fmt.Errorf("store.GetJob: %w", err)
	}
	job, err := toGetJobDTO(row)
	if err != nil {
		return dto.Job{}, fmt.Errorf("store.GetJob: %w", err)
	}
	return job, nil
}

func normalizeJobURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", fmt.Errorf("invalid job URL")
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	return u.String(), nil
}

func jobFingerprint(job dto.Job) string {
	content, _ := json.Marshal([5]string{job.Title, job.Description, job.Location, job.SalaryRaw, job.WorkArrangement})
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func (s *Store) SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error) {
	normalizedURL, err := normalizeJobURL(job.URL)
	if err != nil {
		return dto.Job{}, "", err
	}
	job.URL = normalizedURL
	job.ContentFingerprint = jobFingerprint(job)
	companyID, err := optionalUUID(job.CompanyID)
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("company ID: %w", err)
	}
	boardID, err := optionalUUID(job.BoardID)
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("board ID: %w", err)
	}
	postingID := pgtype.Text{String: job.ProviderPostingID, Valid: job.ProviderPostingID != ""}
	fingerprint := pgtype.Text{String: job.ContentFingerprint, Valid: true}
	updatedAt := pgtype.Timestamptz{Time: job.UpdatedAt, Valid: true}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("begin canonical job: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	identity := normalizedURL
	if boardID.Valid && postingID.Valid {
		identity = job.BoardID + ":" + job.ProviderPostingID
	}
	if err := queries.LockCanonicalJob(ctx, identity); err != nil {
		return dto.Job{}, "", fmt.Errorf("lock canonical job: %w", err)
	}

	previous, err := queries.FindCanonicalJob(ctx, sqlc.FindCanonicalJobParams{
		Url: normalizedURL, BoardID: boardID, PostingID: postingID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return dto.Job{}, "", fmt.Errorf("find canonical job: %w", err)
	}
	if previous.PrimaryBoardID.Valid && boardID.Valid &&
		(previous.PrimaryBoardID != boardID || previous.ProviderPostingID != postingID) {
		return dto.Job{}, "", fmt.Errorf("%w: URL belongs to another trusted posting", ErrCanonicalConflict)
	}

	status := "new"
	var id string
	if errors.Is(err, pgx.ErrNoRows) {
		id, err = queries.InsertCanonicalJob(ctx, sqlc.InsertCanonicalJobParams{
			Title: job.Title, Location: job.Location, Url: job.URL, CompanySlug: job.CompanySlug,
			Source: job.Source, UpdatedAt: updatedAt, Description: job.Description,
			SalaryRaw: job.SalaryRaw, WorkArrangement: job.WorkArrangement,
			CompanyID: companyID, BoardID: boardID, PostingID: postingID, Fingerprint: fingerprint,
		})
		if err != nil {
			return dto.Job{}, "", fmt.Errorf("insert canonical job: %w", err)
		}
	} else {
		id = previous.ID
		oldFingerprint := previous.ContentFingerprint
		if oldFingerprint == "" {
			oldFingerprint = jobFingerprint(dto.Job{
				Title: previous.Title, Description: previous.Description, Location: previous.Location,
				SalaryRaw: previous.SalaryRaw, WorkArrangement: previous.WorkArrangement,
			})
		}
		jobID, err := parseUUID(id)
		if err != nil {
			return dto.Job{}, "", err
		}
		if oldFingerprint == job.ContentFingerprint {
			status = "unchanged"
			err = queries.UpdateUnchangedCanonicalJob(ctx, sqlc.UpdateUnchangedCanonicalJobParams{
				CompanyID: companyID, BoardID: boardID, PostingID: postingID,
				Fingerprint: fingerprint, ID: jobID,
			})
		} else {
			status = "changed"
			err = queries.UpdateChangedCanonicalJob(ctx, sqlc.UpdateChangedCanonicalJobParams{
				Title: job.Title, Location: job.Location, UpdatedAt: updatedAt,
				Description: job.Description, SalaryRaw: job.SalaryRaw,
				WorkArrangement: job.WorkArrangement, Fingerprint: fingerprint,
				CompanyID: companyID, BoardID: boardID, PostingID: postingID, ID: jobID,
			})
		}
		if err != nil {
			return dto.Job{}, "", fmt.Errorf("update canonical job: %w", err)
		}
		job.URL = previous.Url
	}

	jobID, err := parseUUID(id)
	if err != nil {
		return dto.Job{}, "", err
	}
	rows, err := queries.SaveCanonicalJobAlias(ctx, sqlc.SaveCanonicalJobAliasParams{
		Column1: jobID, NormalizedUrl: normalizedURL, Source: job.Source,
	})
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("save job URL: %w", err)
	}
	if rows != 1 {
		return dto.Job{}, "", fmt.Errorf("%w: URL belongs to another canonical job", ErrCanonicalConflict)
	}
	if status != "unchanged" {
		if err := s.scoring.JobsChanged(ctx, tx, []string{id}, status == "new"); err != nil {
			return dto.Job{}, "", fmt.Errorf("scoring.JobsChanged: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.Job{}, "", fmt.Errorf("commit canonical job: %w", err)
	}
	job.ID = id
	return job, status, nil
}

func (s *Store) UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error) {
	row, err := s.queries.UpsertCompany(ctx, sqlc.UpsertCompanyParams{
		Slug:              c.Slug,
		Name:              c.Name,
		AtsSource:         pgtype.Text{String: c.ATSSource, Valid: c.ATSSource != ""},
		AtsToken:          pgtype.Text{String: c.ATSToken, Valid: c.ATSToken != ""},
		Domain:            pgtype.Text{String: c.Domain, Valid: c.Domain != ""},
		LinkedinCompanyID: pgtype.Text{String: c.LinkedInCompanyID, Valid: c.LinkedInCompanyID != ""},
	})
	if err != nil {
		return dto.Company{}, fmt.Errorf("store.UpsertCompany: %w", err)
	}
	return toCompanyDTO(row), nil
}

func (s *Store) GetCompany(ctx context.Context, id string) (dto.Company, error) {
	cid, err := parseUUID(id)
	if err != nil {
		return dto.Company{}, err
	}
	row, err := s.queries.GetCompany(ctx, cid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Company{}, ErrNotFound
	}
	if err != nil {
		return dto.Company{}, fmt.Errorf("store.GetCompany: %w", err)
	}
	return toCompanyDTO(row), nil
}

func (s *Store) ListCompaniesForUser(ctx context.Context, userID string) ([]dto.Company, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListCompaniesForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListCompaniesForUser: %w", err)
	}
	out := make([]dto.Company, len(rows))
	for i, r := range rows {
		out[i] = toCompanyForUserDTO(r)
	}
	return out, nil
}

func (s *Store) SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, interval int) (dto.CompanyTracking, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	cid, err := parseUUID(companyID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("begin company tracking: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	row, err := queries.SetCompanyTracking(ctx, sqlc.SetCompanyTrackingParams{
		UserID: uid, CompanyID: cid, Enabled: enabled, CheckIntervalMinutes: int32(interval),
	})
	if err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("store.SetCompanyTracking: %w", err)
	}
	if enabled {
		if err := queries.BackfillCompanyJobFingerprints(ctx, cid); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("backfill tracked company jobs: %w", err)
		}
		if err := s.scoring.CompanyTracked(ctx, tx, userID, companyID); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("scoring.CompanyTracked: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("commit company tracking: %w", err)
	}
	return dto.CompanyTracking{
		UserID: row.UserID.String(), CompanyID: row.CompanyID.String(),
		Enabled: row.Enabled, CheckIntervalMinutes: int(row.CheckIntervalMinutes),
	}, nil
}

func (s *Store) ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error) {
	id, err := parseUUID(companyID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListCompanyBoards(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("store.ListCompanyBoards: %w", err)
	}
	boards := make([]dto.CompanyBoard, len(rows))
	checks, err := s.queries.ListBoardChecks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("store.ListCompanyBoards checks: %w", err)
	}
	completed := make(map[string]pgtype.Timestamptz, len(checks))
	for _, check := range checks {
		completed[check.ID.String()] = check.LastCompletedAt
	}
	for i, row := range rows {
		boards[i] = toCompanyBoardDTO(row)
		if last := completed[boards[i].ID]; last.Valid {
			boards[i].LastCompletedAt = &last.Time
		}
	}
	return boards, nil
}

func (s *Store) UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error) {
	id, err := parseUUID(companyID)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	row, err := s.queries.UpsertCandidateBoard(ctx, sqlc.UpsertCandidateBoardParams{
		CompanyID: id, Source: source, BoardToken: token,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.CompanyBoard{}, ErrBoardConflict
	}
	if err != nil {
		return dto.CompanyBoard{}, fmt.Errorf("store.UpsertCandidateBoard: %w", err)
	}
	return toCompanyBoardDTO(row), nil
}

func (s *Store) GetVerifiedBoardID(ctx context.Context, source, token string) (string, error) {
	id, err := s.queries.GetVerifiedBoardID(ctx, sqlc.GetVerifiedBoardIDParams{Source: source, BoardToken: token})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store.GetVerifiedBoardID: %w", err)
	}
	return id.String(), nil
}

func toSourceTargetDTO(row sqlc.SourceTarget) dto.SourceTarget {
	filters := map[string]string{}
	if len(row.Filters) > 0 {
		_ = json.Unmarshal(row.Filters, &filters)
	}
	t := dto.SourceTarget{
		ID:                   row.ID.String(),
		UserID:               row.UserID.String(),
		Source:               row.Source,
		Value:                row.Value,
		Enabled:              row.Enabled,
		Filters:              filters,
		CheckIntervalMinutes: int(row.CheckIntervalMinutes),
		RunStatus:            row.RunStatus,
		LastRunError:         row.LastRunError,
		UpdatedAt:            row.UpdatedAt.Time,
	}
	if row.RunID.Valid {
		t.RunID = row.RunID.String()
	}
	if row.CompanyID.Valid {
		t.CompanyID = row.CompanyID.String()
	}
	if row.LastCheckedAt.Valid {
		lc := row.LastCheckedAt.Time
		t.LastCheckedAt = &lc
	}
	if row.LastRunAt.Valid {
		lastRun := row.LastRunAt.Time
		t.LastRunAt = &lastRun
	}
	return t
}

func (s *Store) ListSourceTargetsByUser(ctx context.Context, userID string) ([]dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListSourceTargetsByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListSourceTargetsByUser: %w", err)
	}
	out := make([]dto.SourceTarget, len(rows))
	for i, r := range rows {
		out[i] = toSourceTargetDTO(r)
	}
	return out, nil
}

func (s *Store) CreateSourceTarget(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("store.CreateSourceTarget: marshal filters: %w", err)
	}
	row, err := s.queries.CreateSourceTarget(ctx, sqlc.CreateSourceTargetParams{
		UserID: uid, Source: source, Value: value, Enabled: enabled, Filters: filtersJSON,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.SourceTarget{}, ErrSourceTargetExists
		}
		return dto.SourceTarget{}, fmt.Errorf("store.CreateSourceTarget: %w", err)
	}
	return toSourceTargetDTO(row), nil
}

func (s *Store) CreateSourceTargetWithRun(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := s.queries.CreateSourceTargetWithRun(ctx, sqlc.CreateSourceTargetWithRunParams{
		UserID: uid, Source: source, Value: value, Enabled: enabled, Filters: filtersJSON,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.SourceTarget{}, ErrSourceTargetExists
		}
		return dto.SourceTarget{}, fmt.Errorf("create source target with run: %w", err)
	}
	return toSourceTargetDTO(row), nil
}

func (s *Store) UpsertSourceTargetForCompany(ctx context.Context, userID, source, value, companyID string, enabled bool, interval int) (dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	cid, err := parseUUID(companyID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := s.queries.UpsertSourceTargetForCompany(ctx, sqlc.UpsertSourceTargetForCompanyParams{
		UserID: uid, Source: source, Value: value, Enabled: enabled,
		CompanyID: cid, CheckIntervalMinutes: int32(interval),
	})
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("store.UpsertSourceTargetForCompany: %w", err)
	}
	return toSourceTargetDTO(row), nil
}

func (s *Store) UpdateSourceTarget(ctx context.Context, id, userID string, enabled *bool, checkIntervalMinutes *int) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	params := sqlc.UpdateSourceTargetParams{ID: tid, UserID: uid}
	if enabled != nil {
		params.Enabled = pgtype.Bool{Bool: *enabled, Valid: true}
	}
	if checkIntervalMinutes != nil {
		params.CheckIntervalMinutes = pgtype.Int4{Int32: int32(*checkIntervalMinutes), Valid: true}
	}
	row, err := s.queries.UpdateSourceTarget(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.SourceTarget{}, ErrNotFound
		}
		return dto.SourceTarget{}, fmt.Errorf("store.UpdateSourceTarget: %w", err)
	}
	return toSourceTargetDTO(row), nil
}

func (s *Store) DeleteSourceTarget(ctx context.Context, id, userID string) error {
	tid, err := parseUUID(id)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteSourceTarget(ctx, sqlc.DeleteSourceTargetParams{ID: tid, UserID: uid}); err != nil {
		return fmt.Errorf("store.DeleteSourceTarget: %w", err)
	}
	return nil
}

func (s *Store) StartSourceTargetRun(ctx context.Context, id string) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := s.queries.StartSourceTargetRun(ctx, tid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.SourceTarget{}, ErrNotFound
	}
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("start source target run: %w", err)
	}
	return toSourceTargetDTO(row), nil
}

func normalizedCandidateURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid candidate URL %q", raw)
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String(), nil
}

func (s *Store) SaveCards(ctx context.Context, target dto.SourceTarget, cards []dto.Job) ([]candidates.Candidate, error) {
	targetID, err := parseUUID(target.ID)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin candidate save: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	out := make([]candidates.Candidate, 0, len(cards))
	for _, card := range cards {
		if card.URL == "" {
			continue
		}
		normalized, err := normalizedCandidateURL(card.URL)
		if err != nil {
			return nil, err
		}
		id, err := queries.UpsertCandidate(ctx, sqlc.UpsertCandidateParams{
			NormalizedUrl: normalized, Source: target.Source,
			CardTitle: card.Title, CardCompany: card.CompanySlug, CardLocation: card.Location,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert candidate: %w", err)
		}
		if err := queries.RecordCandidateDiscovery(ctx, sqlc.RecordCandidateDiscoveryParams{
			CandidateID: id, SourceTargetID: targetID,
		}); err != nil {
			return nil, fmt.Errorf("record candidate discovery: %w", err)
		}
		card.URL = normalized
		card.Source = target.Source
		out = append(out, candidates.Candidate{ID: id.String(), URL: normalized, Card: card})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit candidate save: %w", err)
	}
	return out, nil
}

func (s *Store) ListForUser(ctx context.Context, userID, afterID string, limit int) ([]candidates.Candidate, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("candidate batch limit out of range: %d", limit)
	}
	if afterID == "" {
		afterID = "00000000-0000-0000-0000-000000000000"
	}
	after, err := parseUUID(afterID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListCandidatesForUser(ctx, sqlc.ListCandidatesForUserParams{
		UserID: uid, ID: after, Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("query candidates: %w", err)
	}
	out := make([]candidates.Candidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, candidates.Candidate{
			ID: row.ID.String(), URL: row.NormalizedUrl,
			Card: dto.Job{URL: row.NormalizedUrl, Title: row.CardTitle, CompanySlug: row.CardCompany,
				Location: row.CardLocation, Source: row.Source},
		})
	}
	return out, nil
}

func (s *Store) Assess(ctx context.Context, candidateID, userID string, version time.Time, passes bool) (bool, error) {
	cid, err := parseUUID(candidateID)
	if err != nil {
		return false, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return false, err
	}
	claimed, err := s.queries.AssessCandidate(ctx, sqlc.AssessCandidateParams{
		CandidateID: cid, UserID: uid,
		SearchConfigVersion: pgtype.Timestamptz{Time: version, Valid: true},
		Relevance:           passes,
	})
	if err != nil {
		return false, fmt.Errorf("assess candidate: %w", err)
	}
	return claimed, nil
}

func (s *Store) MarkDetailPending(ctx context.Context, candidateID string) error {
	cid, err := parseUUID(candidateID)
	if err != nil {
		return err
	}
	return s.queries.MarkCandidateDetailPending(ctx, cid)
}
