package db

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

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

func (db *DB) SaveCards(ctx context.Context, target dto.SourceTarget, cards []dto.Job) ([]candidates.Candidate, error) {
	targetID, err := parseUUID(target.ID)
	if err != nil {
		return nil, err
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin candidate save: %w", err)
	}
	defer tx.Rollback(ctx)

	out := make([]candidates.Candidate, 0, len(cards))
	for _, card := range cards {
		if card.URL == "" {
			continue
		}
		normalized, err := normalizedCandidateURL(card.URL)
		if err != nil {
			return nil, err
		}
		var id string
		err = tx.QueryRow(ctx, `
INSERT INTO job_candidates (normalized_url, source, card_title, card_company, card_location)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (normalized_url) DO UPDATE SET
    source = EXCLUDED.source,
    card_title = EXCLUDED.card_title,
    card_company = EXCLUDED.card_company,
    card_location = EXCLUDED.card_location,
    last_seen_at = NOW(),
    expires_at = NOW() + INTERVAL '60 days'
RETURNING id`, normalized, target.Source, card.Title, card.CompanySlug, card.Location).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert candidate: %w", err)
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO candidate_discoveries (candidate_id, source_target_id)
VALUES ($1, $2)
ON CONFLICT (candidate_id, source_target_id) DO UPDATE SET last_seen_at = NOW()`, id, targetID); err != nil {
			return nil, fmt.Errorf("record candidate discovery: %w", err)
		}
		card.URL = normalized
		out = append(out, candidates.Candidate{ID: id, URL: normalized, Card: card})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit candidate save: %w", err)
	}
	return out, nil
}

func (db *DB) ListForUser(ctx context.Context, userID, afterID string, limit int) ([]candidates.Candidate, error) {
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
	rows, err := db.pool.Query(ctx, `
SELECT c.id, c.normalized_url, c.card_title, c.card_company, c.card_location, c.source
FROM job_candidates c
WHERE c.id > $2 AND c.expires_at > NOW()
  AND EXISTS (
      SELECT 1 FROM candidate_discoveries d
      JOIN source_targets t ON t.id = d.source_target_id
      WHERE d.candidate_id = c.id AND t.user_id = $1 AND t.enabled
  )
ORDER BY c.id
LIMIT $3`, uid, after, limit)
	if err != nil {
		return nil, fmt.Errorf("query candidates: %w", err)
	}
	defer rows.Close()
	out := make([]candidates.Candidate, 0, limit)
	for rows.Next() {
		var candidate candidates.Candidate
		if err := rows.Scan(&candidate.ID, &candidate.URL, &candidate.Card.Title, &candidate.Card.CompanySlug, &candidate.Card.Location, &candidate.Card.Source); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		candidate.Card.URL = candidate.URL
		out = append(out, candidate)
	}
	return out, rows.Err()
}

func (db *DB) Assess(ctx context.Context, candidateID, userID string, version time.Time, passes bool) (bool, error) {
	cid, err := parseUUID(candidateID)
	if err != nil {
		return false, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return false, err
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin candidate assessment: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
INSERT INTO candidate_assessments (candidate_id, user_id, search_config_version, relevance)
VALUES ($1, $2, $3, $4)
ON CONFLICT (candidate_id, user_id) DO UPDATE SET
    search_config_version = EXCLUDED.search_config_version,
    relevance = EXCLUDED.relevance,
    evaluated_at = NOW()`, cid, uid, version, passes); err != nil {
		return false, fmt.Errorf("record candidate assessment: %w", err)
	}
	queueDetail := false
	if passes {
		err = tx.QueryRow(ctx, `
UPDATE job_candidates c SET detail_state = 'pending'
WHERE c.id = $1 AND c.detail_state = 'unrequested' AND c.expires_at > NOW()
  AND NOT EXISTS (SELECT 1 FROM jobs j WHERE j.url = c.normalized_url)
RETURNING true`, cid).Scan(&queueDetail)
		if err != nil && err != pgx.ErrNoRows {
			return false, fmt.Errorf("claim candidate detail: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit candidate assessment: %w", err)
	}
	return queueDetail, nil
}

func (db *DB) ReleaseDetail(ctx context.Context, candidateID string) error {
	cid, err := parseUUID(candidateID)
	if err != nil {
		return err
	}
	_, err = db.pool.Exec(ctx, `UPDATE job_candidates SET detail_state = 'unrequested' WHERE id = $1`, cid)
	return err
}

func (db *DB) DeleteExpiredCandidates(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM job_candidates WHERE expires_at <= NOW()`)
	return err
}
