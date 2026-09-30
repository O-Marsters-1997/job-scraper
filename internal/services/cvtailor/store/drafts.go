package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

var (
	ErrJobNotFound   = apperr.NotFound("job not found")
	ErrDraftNotFound = apperr.NotFound("draft not found")
)

var ErrKeptDraftExists = apperr.Conflict("this job already has a kept draft")

func toDraft(t sqlc.TailoredCv) (dto.Draft, error) {
	findings := []dto.DraftFinding{}
	if err := json.Unmarshal(t.Findings, &findings); err != nil {
		return dto.Draft{}, fmt.Errorf("decode findings: %w", err)
	}
	var outcome *string
	if t.Outcome.Valid {
		outcome = &t.Outcome.String
	}
	return dto.Draft{
		ID: t.ID.String(), JobID: t.JobID.String(), Status: t.Status, Outcome: outcome, LastError: t.LastError,
		CreatedAt: t.CreatedAt.Time, Findings: findings, DraftDocID: t.DraftDocID.String, EditSet: t.EditSet,
	}, nil
}

// CreateDraft inserts a pending Draft for userID; ErrJobNotFound when the Job
// does not exist.
func (s *Store) CreateDraft(ctx context.Context, userID string, in dto.DraftInput) (dto.Draft, error) {
	uid, err := parseID(userID, ErrDraftNotFound)
	if err != nil {
		return dto.Draft{}, err
	}
	jid, err := parseID(in.JobID, ErrJobNotFound)
	if err != nil {
		return dto.Draft{}, err
	}
	achievements, err := data.UUIDs(in.AchievementIDs)
	if err != nil {
		return dto.Draft{}, apperr.Invalid("unknown achievement")
	}
	row, err := s.queries.InsertDraft(ctx, sqlc.InsertDraftParams{
		UserID: uid, JobID: jid, BaseDocID: in.DocID, BaseTabID: in.TabID, AchievementIds: achievements,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Draft{}, ErrJobNotFound
	}
	if err != nil {
		return dto.Draft{}, fmt.Errorf("store.CreateDraft: %w", err)
	}
	return toDraft(row)
}

func (s *Store) GetDraft(ctx context.Context, userID, id string) (dto.Draft, error) {
	uid, err := parseID(userID, ErrDraftNotFound)
	if err != nil {
		return dto.Draft{}, err
	}
	did, err := parseID(id, ErrDraftNotFound)
	if err != nil {
		return dto.Draft{}, err
	}
	row, err := s.queries.GetDraft(ctx, sqlc.GetDraftParams{ID: did, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Draft{}, ErrDraftNotFound
	}
	if err != nil {
		return dto.Draft{}, fmt.Errorf("store.GetDraft: %w", err)
	}
	return toDraft(row)
}

// ListJobDrafts returns userID's Drafts of the Job, newest first.
func (s *Store) ListJobDrafts(ctx context.Context, userID, jobID string) ([]dto.Draft, error) {
	uid, err := parseID(userID, ErrDraftNotFound)
	if err != nil {
		return nil, err
	}
	jid, err := parseID(jobID, ErrJobNotFound)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListJobDrafts(ctx, sqlc.ListJobDraftsParams{JobID: jid, UserID: uid})
	if err != nil {
		return nil, fmt.Errorf("store.ListJobDrafts: %w", err)
	}
	out := make([]dto.Draft, len(rows))
	for i, row := range rows {
		if out[i], err = toDraft(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// SetDraftOutcome records outcome on the Draft. Discarding also drops its
// Doc id. Keeping a second Draft of one Job is ErrKeptDraftExists.
func (s *Store) SetDraftOutcome(ctx context.Context, userID, id, outcome string) (dto.Draft, error) {
	uid, err := parseID(userID, ErrDraftNotFound)
	if err != nil {
		return dto.Draft{}, err
	}
	did, err := parseID(id, ErrDraftNotFound)
	if err != nil {
		return dto.Draft{}, err
	}
	row, err := s.queries.SetDraftOutcome(ctx, sqlc.SetDraftOutcomeParams{Outcome: outcome, ID: did, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Draft{}, ErrDraftNotFound
	}
	if data.IsUniqueViolation(err) {
		return dto.Draft{}, ErrKeptDraftExists
	}
	if err != nil {
		return dto.Draft{}, fmt.Errorf("store.SetDraftOutcome: %w", err)
	}
	return toDraft(row)
}

// ClaimDraft leases the next due Draft, or one whose lease expired, under
// FOR UPDATE SKIP LOCKED; data.ErrNotFound when none is due.
func (s *Store) ClaimDraft(ctx context.Context) (dto.DraftClaim, error) {
	row, err := s.queries.ClaimDraft(ctx)
	if err != nil {
		return dto.DraftClaim{}, data.QueryErr("ClaimDraft", err)
	}
	ids := make([]string, len(row.AchievementIds))
	for i, id := range row.AchievementIds {
		ids[i] = id.String()
	}
	return dto.DraftClaim{
		ID: row.ID.String(), UserID: row.UserID.String(), JobID: row.JobID.String(),
		DocID: row.BaseDocID, TabID: row.BaseTabID, AchievementIDs: ids,
		Attempts: int(row.Attempts), DraftDocID: row.DraftDocID,
		JobDescription: row.JobDescription, JobFingerprint: row.JobFingerprint,
	}, nil
}

func (s *Store) SetDraftDoc(ctx context.Context, claim dto.DraftClaim, docID string) error {
	id, err := parseID(claim.ID, ErrDraftNotFound)
	if err != nil {
		return err
	}
	err = s.queries.SetDraftDoc(ctx, sqlc.SetDraftDocParams{DraftDocID: docID, ID: id, Attempts: int32(claim.Attempts)})
	if err != nil {
		return fmt.Errorf("store.SetDraftDoc: %w", err)
	}
	return nil
}

// CompleteDraft marks the claimed Draft ready; a stale claim (the lease was
// re-claimed meanwhile) is ErrDraftNotFound.
func (s *Store) CompleteDraft(ctx context.Context, claim dto.DraftClaim, res dto.DraftResult) error {
	id, err := parseID(claim.ID, ErrDraftNotFound)
	if err != nil {
		return err
	}
	findings, err := json.Marshal(res.Findings)
	if err != nil {
		return fmt.Errorf("store.CompleteDraft: encode findings: %w", err)
	}
	n, err := s.queries.CompleteDraft(ctx, sqlc.CompleteDraftParams{
		ID: id, Attempts: int32(claim.Attempts), EditSet: res.EditSet, RawOutput: res.RawOutput,
		Model: res.Model, PromptVersion: res.PromptVersion, JobFingerprint: res.JobFingerprint,
		Cost: float32(res.Cost), DraftDocID: res.DraftDocID, Findings: findings,
	})
	if err != nil {
		return fmt.Errorf("store.CompleteDraft: %w", err)
	}
	if n == 0 {
		return ErrDraftNotFound
	}
	return nil
}

// FailDraft releases the claimed Draft for a backed-off retry, or marks it
// failed when terminal or out of attempts.
func (s *Store) FailDraft(ctx context.Context, claim dto.DraftClaim, failure dto.DraftFailure) error {
	id, err := parseID(claim.ID, ErrDraftNotFound)
	if err != nil {
		return err
	}
	n, err := s.queries.FailDraft(ctx, sqlc.FailDraftParams{
		ID: id, Attempts: int32(claim.Attempts), Terminal: failure.Terminal, MaxAttempts: dto.MaxDraftAttempts,
		LastError: failure.Reason, ClearDoc: failure.ClearDoc,
	})
	if err != nil {
		return fmt.Errorf("store.FailDraft: %w", err)
	}
	if n == 0 {
		return ErrDraftNotFound
	}
	return nil
}
