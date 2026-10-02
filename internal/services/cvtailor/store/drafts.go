package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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
	achievementIDs := make([]string, len(t.AchievementIds))
	for i, id := range t.AchievementIds {
		achievementIDs[i] = id.String()
	}
	var outcome *string
	if t.Outcome.Valid {
		outcome = &t.Outcome.String
	}
	return dto.Draft{
		ID: t.ID.String(), JobID: t.JobID.String(), Status: t.Status, Outcome: outcome, KeptAs: t.KeptAs.String, LastError: t.LastError,
		CreatedAt: t.CreatedAt.Time, Findings: findings, DraftDocID: t.DraftDocID.String, EditSet: t.EditSet, BaseContent: t.BaseContent,
		BaseDocID: t.BaseDocID, BaseTabID: t.BaseTabID, AchievementIDs: achievementIDs,
	}, nil
}

// CreateDraft inserts a pending Draft for userID with its bullet labels;
// ErrJobNotFound when the Job does not exist.
func (s *Store) CreateDraft(ctx context.Context, userID string, in dto.DraftInput, labels []dto.BulletLabel) (dto.Draft, error) {
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
	var row sqlc.TailoredCv
	err = s.inTx(ctx, func(q *sqlc.Queries) error {
		var err error
		row, err = q.InsertDraft(ctx, sqlc.InsertDraftParams{
			UserID: uid, JobID: jid, BaseDocID: in.DocID, BaseTabID: in.TabID, AchievementIds: achievements,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrJobNotFound
		}
		if err != nil {
			return fmt.Errorf("store.CreateDraft: %w", err)
		}
		params, err := bulletLabelParams(uid, jid, row.ID, labels)
		if err != nil {
			return fmt.Errorf("store.CreateDraft: %w", err)
		}
		if _, err := q.InsertBulletLabels(ctx, params); err != nil {
			return fmt.Errorf("store.CreateDraft: labels: %w", err)
		}
		return nil
	})
	if err != nil {
		return dto.Draft{}, err
	}
	return toDraft(row)
}

func bulletLabelParams(userID, jobID, draftID pgtype.UUID, labels []dto.BulletLabel) ([]sqlc.InsertBulletLabelsParams, error) {
	params := make([]sqlc.InsertBulletLabelsParams, len(labels))
	for i, l := range labels {
		aid, err := data.UUID(l.AchievementID)
		if err != nil {
			return nil, err
		}
		params[i] = sqlc.InsertBulletLabelsParams{
			UserID: userID, JobID: jobID, DraftID: draftID, Kind: "bullet", AchievementID: aid,
			Preselected: l.Preselected, Kept: l.Kept,
		}
		if a := l.Answer; a != nil {
			params[i].PYes = float8(a.PYes)
			params[i].PNo = float8(a.PNo)
			params[i].PNotStated = float8(a.PNotStated)
			params[i].Confidence = float8(a.Confidence)
		}
	}
	return params, nil
}

func float8(f float64) pgtype.Float8 {
	return pgtype.Float8{Float64: f, Valid: true}
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
		Keeping: row.Keeping, JobTitle: row.JobTitle, CompanyName: row.CompanyName,
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
		ID: id, Attempts: int32(claim.Attempts), EditSet: res.EditSet, BaseContent: res.BaseContent, RawOutput: res.RawOutput,
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

// QueueKeep moves a ready, undecided Draft to keeping; ErrDraftNotFound when
// it is missing, another User's or no longer ready and undecided.
func (s *Store) QueueKeep(ctx context.Context, userID, id string) (dto.Draft, error) {
	uid, err := parseID(userID, ErrDraftNotFound)
	if err != nil {
		return dto.Draft{}, err
	}
	did, err := parseID(id, ErrDraftNotFound)
	if err != nil {
		return dto.Draft{}, err
	}
	row, err := s.queries.QueueKeep(ctx, sqlc.QueueKeepParams{ID: did, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Draft{}, ErrDraftNotFound
	}
	if err != nil {
		return dto.Draft{}, fmt.Errorf("store.QueueKeep: %w", err)
	}
	return toDraft(row)
}

// CompleteKeep marks the claimed Draft kept as a Doc. A stale claim is
// ErrDraftNotFound; a second kept Draft of the Job is ErrKeptDraftExists.
func (s *Store) CompleteKeep(ctx context.Context, claim dto.DraftClaim) error {
	id, err := parseID(claim.ID, ErrDraftNotFound)
	if err != nil {
		return err
	}
	n, err := s.queries.CompleteKeep(ctx, sqlc.CompleteKeepParams{ID: id, Attempts: int32(claim.Attempts)})
	if data.IsUniqueViolation(err) {
		return ErrKeptDraftExists
	}
	if err != nil {
		return fmt.Errorf("store.CompleteKeep: %w", err)
	}
	if n == 0 {
		return ErrDraftNotFound
	}
	return nil
}

// FailKeep releases the claimed Draft for a backed-off retry, or returns it
// to ready, unkept, when terminal or out of attempts.
func (s *Store) FailKeep(ctx context.Context, claim dto.DraftClaim, failure dto.DraftFailure) error {
	id, err := parseID(claim.ID, ErrDraftNotFound)
	if err != nil {
		return err
	}
	n, err := s.queries.FailKeep(ctx, sqlc.FailKeepParams{
		ID: id, Attempts: int32(claim.Attempts), Terminal: failure.Terminal, MaxAttempts: dto.MaxDraftAttempts,
		LastError: failure.Reason,
	})
	if err != nil {
		return fmt.Errorf("store.FailKeep: %w", err)
	}
	if n == 0 {
		return ErrDraftNotFound
	}
	return nil
}

// SetDraftEdits replaces the Draft's edit set and findings after the User's
// edits; ErrDraftNotFound when userID does not own it.
func (s *Store) SetDraftEdits(ctx context.Context, userID, id string, editSet json.RawMessage, findings []dto.DraftFinding) error {
	uid, err := parseID(userID, ErrDraftNotFound)
	if err != nil {
		return err
	}
	did, err := parseID(id, ErrDraftNotFound)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(findings)
	if err != nil {
		return fmt.Errorf("store.SetDraftEdits: encode findings: %w", err)
	}
	n, err := s.queries.SetDraftEdits(ctx, sqlc.SetDraftEditsParams{ID: did, UserID: uid, EditSet: editSet, Findings: raw})
	if err != nil {
		return fmt.Errorf("store.SetDraftEdits: %w", err)
	}
	if n == 0 {
		return ErrDraftNotFound
	}
	return nil
}

func (s *Store) JobDescription(ctx context.Context, jobID string) (string, error) {
	id, err := parseID(jobID, ErrJobNotFound)
	if err != nil {
		return "", err
	}
	desc, err := s.queries.GetJobDescription(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrJobNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store.JobDescription: %w", err)
	}
	return desc, nil
}
