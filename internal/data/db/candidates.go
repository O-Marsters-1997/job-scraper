package db

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
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
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.queries.WithTx(tx)

	out := make([]candidates.Candidate, 0, len(cards))
	for _, card := range cards {
		if card.URL == "" {
			continue
		}
		normalized, err := normalizedCandidateURL(card.URL)
		if err != nil {
			return nil, err
		}
		id, err := queries.UpsertCandidate(ctx, pgsqlc.UpsertCandidateParams{
			NormalizedUrl: normalized,
			Source:        target.Source,
			CardTitle:     card.Title,
			CardCompany:   card.CompanySlug,
			CardLocation:  card.Location,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert candidate: %w", err)
		}
		if err := queries.RecordCandidateDiscovery(ctx, pgsqlc.RecordCandidateDiscoveryParams{
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
	rows, err := db.queries.ListCandidatesForUser(ctx, pgsqlc.ListCandidatesForUserParams{
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

func (db *DB) Assess(ctx context.Context, candidateID, userID string, version time.Time, passes bool) (bool, error) {
	cid, err := parseUUID(candidateID)
	if err != nil {
		return false, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return false, err
	}
	claimed, err := db.queries.AssessCandidate(ctx, pgsqlc.AssessCandidateParams{
		CandidateID: cid, UserID: uid,
		SearchConfigVersion: pgtype.Timestamptz{Time: version, Valid: true},
		Relevance:           passes,
	})
	if err != nil {
		return false, fmt.Errorf("assess candidate: %w", err)
	}
	return claimed, nil
}

func (db *DB) ReleaseDetail(ctx context.Context, candidateID string) error {
	cid, err := parseUUID(candidateID)
	if err != nil {
		return err
	}
	return db.queries.ReleaseCandidateDetail(ctx, cid)
}

func (db *DB) MarkDetailPending(ctx context.Context, candidateID string) error {
	cid, err := parseUUID(candidateID)
	if err != nil {
		return err
	}
	return db.queries.MarkCandidateDetailPending(ctx, cid)
}

func (db *DB) DeleteExpiredCandidates(ctx context.Context) error {
	return db.queries.DeleteExpiredCandidates(ctx)
}
