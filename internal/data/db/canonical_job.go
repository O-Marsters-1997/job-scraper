package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
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
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (db *DB) SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error) {
	normalizedURL, err := normalizeJobURL(job.URL)
	if err != nil {
		return dto.Job{}, "", err
	}
	job.URL = normalizedURL
	job.ContentFingerprint = jobFingerprint(job)

	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("begin canonical job: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	identity := normalizedURL
	if job.BoardID != "" && job.ProviderPostingID != "" {
		identity = job.BoardID + ":" + job.ProviderPostingID
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", identity); err != nil {
		return dto.Job{}, "", fmt.Errorf("lock canonical job: %w", err)
	}

	var id, canonicalURL, oldFingerprint, oldTitle, oldDescription, oldLocation, oldSalary, oldArrangement, oldBoardID, oldPostingID string
	query := `SELECT id::text, url, COALESCE(content_fingerprint, ''), title, description, location, salary_raw, work_arrangement,
        COALESCE(primary_board_id::text, ''), COALESCE(provider_posting_id, '')
        FROM jobs WHERE primary_board_id = NULLIF($1, '')::uuid AND provider_posting_id = NULLIF($2, '') FOR UPDATE`
	if job.BoardID != "" && job.ProviderPostingID != "" {
		err = tx.QueryRow(ctx, query, job.BoardID, job.ProviderPostingID).Scan(&id, &canonicalURL, &oldFingerprint, &oldTitle, &oldDescription, &oldLocation, &oldSalary, &oldArrangement, &oldBoardID, &oldPostingID)
	} else {
		err = pgx.ErrNoRows
	}
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `SELECT j.id::text, j.url, COALESCE(j.content_fingerprint, ''), j.title, j.description, j.location, j.salary_raw, j.work_arrangement,
            COALESCE(j.primary_board_id::text, ''), COALESCE(j.provider_posting_id, '')
            FROM jobs j JOIN job_urls u ON u.job_id = j.id WHERE u.normalized_url = $1 FOR UPDATE OF j`, normalizedURL).
			Scan(&id, &canonicalURL, &oldFingerprint, &oldTitle, &oldDescription, &oldLocation, &oldSalary, &oldArrangement, &oldBoardID, &oldPostingID)
	}
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `SELECT id::text, url, COALESCE(content_fingerprint, ''), title, description, location, salary_raw, work_arrangement,
            COALESCE(primary_board_id::text, ''), COALESCE(provider_posting_id, '')
            FROM jobs WHERE url = $1 FOR UPDATE`, normalizedURL).
			Scan(&id, &canonicalURL, &oldFingerprint, &oldTitle, &oldDescription, &oldLocation, &oldSalary, &oldArrangement, &oldBoardID, &oldPostingID)
	}
	if err != nil && err != pgx.ErrNoRows {
		return dto.Job{}, "", fmt.Errorf("find canonical job: %w", err)
	}
	if oldBoardID != "" && job.BoardID != "" && (oldBoardID != job.BoardID || oldPostingID != job.ProviderPostingID) {
		return dto.Job{}, "", fmt.Errorf("%w: URL belongs to another trusted posting", providers.ErrCanonicalConflict)
	}

	status := "new"
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `INSERT INTO jobs
            (title, location, url, company_slug, source, updated_at, description, salary_raw, work_arrangement,
             company_id, primary_board_id, provider_posting_id, content_fingerprint, content_changed_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid,NULLIF($11,'')::uuid,NULLIF($12,''),$13,NOW())
            RETURNING id::text`, job.Title, job.Location, job.URL, job.CompanySlug, job.Source,
			job.UpdatedAt, job.Description, job.SalaryRaw, job.WorkArrangement, job.CompanyID, job.BoardID,
			job.ProviderPostingID, job.ContentFingerprint).Scan(&id)
		if err != nil {
			return dto.Job{}, "", fmt.Errorf("insert canonical job: %w", err)
		}
	} else {
		previous := dto.Job{Title: oldTitle, Description: oldDescription, Location: oldLocation, SalaryRaw: oldSalary, WorkArrangement: oldArrangement}
		if oldFingerprint == "" {
			oldFingerprint = jobFingerprint(previous)
		}
		status = "unchanged"
		if oldFingerprint != job.ContentFingerprint {
			status = "changed"
			_, err = tx.Exec(ctx, `UPDATE jobs SET title=$2, location=$3, updated_at=$4, description=$5,
                salary_raw=$6, work_arrangement=$7, content_fingerprint=$8, content_changed_at=NOW(),
                company_id=COALESCE(NULLIF($9,'')::uuid, company_id),
                primary_board_id=COALESCE(NULLIF($10,'')::uuid, primary_board_id),
                provider_posting_id=COALESCE(NULLIF($11,''), provider_posting_id), scraped_at=NOW()
                WHERE id=$1::uuid`, id, job.Title, job.Location, job.UpdatedAt, job.Description,
				job.SalaryRaw, job.WorkArrangement, job.ContentFingerprint, job.CompanyID, job.BoardID, job.ProviderPostingID)
		} else {
			_, err = tx.Exec(ctx, `UPDATE jobs SET company_id=COALESCE(NULLIF($2,'')::uuid, company_id),
                primary_board_id=COALESCE(NULLIF($3,'')::uuid, primary_board_id),
				provider_posting_id=COALESCE(NULLIF($4,''), provider_posting_id),
				content_fingerprint=COALESCE(content_fingerprint, $5)
				WHERE id=$1::uuid`, id, job.CompanyID, job.BoardID, job.ProviderPostingID, job.ContentFingerprint)
		}
		if err != nil {
			return dto.Job{}, "", fmt.Errorf("update canonical job: %w", err)
		}
		job.URL = canonicalURL
	}

	command, err := tx.Exec(ctx, `INSERT INTO job_urls(job_id, normalized_url, source) VALUES ($1::uuid,$2,$3)
        ON CONFLICT (normalized_url) DO UPDATE SET last_seen_at=NOW() WHERE job_urls.job_id=EXCLUDED.job_id`, id, normalizedURL, job.Source)
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("save job URL: %w", err)
	}
	if command.RowsAffected() != 1 {
		return dto.Job{}, "", fmt.Errorf("%w: URL belongs to another canonical job", providers.ErrCanonicalConflict)
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.Job{}, "", fmt.Errorf("commit canonical job: %w", err)
	}
	job.ID = id
	return job, status, nil
}
