package db

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

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

func optionalUUID(id string) (pgtype.UUID, error) {
	if id == "" {
		return pgtype.UUID{}, nil
	}
	return parseUUID(id)
}

func (db *DB) SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error) {
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

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("begin canonical job: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.queries.WithTx(tx)

	identity := normalizedURL
	if boardID.Valid && postingID.Valid {
		identity = job.BoardID + ":" + job.ProviderPostingID
	}
	if err := queries.LockCanonicalJob(ctx, identity); err != nil {
		return dto.Job{}, "", fmt.Errorf("lock canonical job: %w", err)
	}

	previous, err := queries.FindCanonicalJob(ctx, pgsqlc.FindCanonicalJobParams{
		Url: normalizedURL, BoardID: boardID, PostingID: postingID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return dto.Job{}, "", fmt.Errorf("find canonical job: %w", err)
	}
	if previous.PrimaryBoardID.Valid && boardID.Valid &&
		(previous.PrimaryBoardID != boardID || previous.ProviderPostingID != postingID) {
		return dto.Job{}, "", fmt.Errorf("%w: URL belongs to another trusted posting", providers.ErrCanonicalConflict)
	}

	status := "new"
	var id string
	if errors.Is(err, pgx.ErrNoRows) {
		id, err = queries.InsertCanonicalJob(ctx, pgsqlc.InsertCanonicalJobParams{
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
			err = queries.UpdateUnchangedCanonicalJob(ctx, pgsqlc.UpdateUnchangedCanonicalJobParams{
				CompanyID: companyID, BoardID: boardID, PostingID: postingID,
				Fingerprint: fingerprint, ID: jobID,
			})
		} else {
			status = "changed"
			err = queries.UpdateChangedCanonicalJob(ctx, pgsqlc.UpdateChangedCanonicalJobParams{
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
	rows, err := queries.SaveCanonicalJobAlias(ctx, pgsqlc.SaveCanonicalJobAliasParams{
		Column1: jobID, NormalizedUrl: normalizedURL, Source: job.Source,
	})
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("save job URL: %w", err)
	}
	if rows != 1 {
		return dto.Job{}, "", fmt.Errorf("%w: URL belongs to another canonical job", providers.ErrCanonicalConflict)
	}
	if status != "unchanged" {
		role, _ := sourcespec.SourceRole(job.Source)
		in := scoringEffectsInput{
			Job: job, JobID: jobID, CompanyID: companyID,
			Discovery: role == sourcespec.RoleDiscovery, FirstDiscovery: status == "new",
		}
		if err := queueScoringEffects(ctx, queries, in); err != nil {
			return dto.Job{}, "", fmt.Errorf("queue scoring effects: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.Job{}, "", fmt.Errorf("commit canonical job: %w", err)
	}
	job.ID = id
	return job, status, nil
}
