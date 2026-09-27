package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var _ providers.SuitabilityProvider = (*DB)(nil)

func queueAnswerEffect(ctx context.Context, queries *pgsqlc.Queries, jobID pgtype.UUID, fingerprint string, discovery, firstDiscovery bool) error {
	return queries.QueueAnswerEffect(ctx, pgsqlc.QueueAnswerEffectParams{
		JobID: jobID, Fingerprint: fingerprint, Discovery: discovery, FirstDiscovery: firstDiscovery,
	})
}

func (db *DB) QueueTrackingScores(ctx context.Context, companyID string) error {
	cid, err := parseUUID(companyID)
	if err != nil {
		return err
	}
	if err := db.queries.QueueTrackingScores(ctx, cid); err != nil {
		return fmt.Errorf("db.QueueTrackingScores: %w", err)
	}
	return nil
}

func (db *DB) ClaimAnswerEffect(ctx context.Context) (dto.AnswerEffect, error) {
	row, err := db.queries.ClaimAnswerEffect(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.AnswerEffect{}, data.ErrNotFound
	}
	if err != nil {
		return dto.AnswerEffect{}, fmt.Errorf("db.ClaimAnswerEffect: %w", err)
	}
	return dto.AnswerEffect{
		ID: row.ID.String(), JobID: row.JobID.String(), Fingerprint: row.Fingerprint,
		Model: row.Model, Attempts: int(row.Attempts), FirstDiscovery: row.FirstDiscovery,
	}, nil
}

func (db *DB) FailAnswerEffect(ctx context.Context, id string, attempts int, failure dto.ScoringFailure) error {
	effectID, err := parseUUID(id)
	if err != nil {
		return err
	}
	var retryAfterSecs pgtype.Int4
	if failure.RetryAfter > 0 {
		retryAfterSecs = pgtype.Int4{Int32: int32(failure.RetryAfter.Seconds()), Valid: true}
	}
	err = db.queries.FailAnswerEffect(ctx, pgsqlc.FailAnswerEffectParams{
		ID: effectID, Attempts: int32(attempts), LastError: failure.Reason,
		Terminal: failure.Terminal, RetryAfterSecs: retryAfterSecs,
	})
	if err != nil {
		return fmt.Errorf("db.FailAnswerEffect: %w", err)
	}
	return nil
}

func (db *DB) GetJobForScoring(ctx context.Context, jobID string) (dto.Job, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return dto.Job{}, providers.ErrInvalidID
	}
	row, err := db.queries.GetJobForScoring(ctx, jid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Job{}, data.ErrNotFound
	}
	if err != nil {
		return dto.Job{}, fmt.Errorf("db.GetJobForScoring: %w", err)
	}
	return dto.Job{
		ID: row.ID.String(), Title: row.Title, Location: row.Location, URL: row.Url,
		CompanySlug: row.CompanySlug, Source: row.Source, UpdatedAt: row.UpdatedAt.Time,
		ScrapedAt: row.ScrapedAt.Time, Description: row.Description, SalaryRaw: row.SalaryRaw,
		WorkArrangement: row.WorkArrangement, CompanyID: uuidString(row.CompanyID),
		BoardID: uuidString(row.PrimaryBoardID), ProviderPostingID: row.ProviderPostingID.String,
		ContentFingerprint: row.ContentFingerprint.String,
	}, nil
}

func (db *DB) ListInterestedConfigs(ctx context.Context, jobID string, discovery bool) ([]dto.SearchConfig, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return nil, providers.ErrInvalidID
	}
	rows, err := db.queries.ListInterestedConfigs(ctx, pgsqlc.ListInterestedConfigsParams{JobID: jid, Discovery: discovery})
	if err != nil {
		return nil, fmt.Errorf("db.ListInterestedConfigs: %w", err)
	}
	configs := make([]dto.SearchConfig, len(rows))
	for i, row := range rows {
		var prefs dto.Preferences
		if err := json.Unmarshal(row.Preferences, &prefs); err != nil {
			return nil, fmt.Errorf("db.ListInterestedConfigs: unmarshal preferences: %w", err)
		}
		configs[i] = dto.SearchConfig{
			UserID:                row.UserID.String(),
			ExcludedTitleKeywords: row.ExcludedTitleKeywords,
			ExcludedCompanies:     row.ExcludedCompanies,
			ExcludedLocations:     row.ExcludedLocations,
			NotifyThreshold:       int(row.NotifyThreshold),
			Preferences:           prefs,
		}
	}
	return configs, nil
}

func (db *DB) ListAnswers(ctx context.Context, jobID, fingerprint, model string) (map[string]dto.Answer, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return nil, providers.ErrInvalidID
	}
	rows, err := db.queries.ListOptionAnswers(ctx, pgsqlc.ListOptionAnswersParams{JobID: jid, Fingerprint: fingerprint, Model: model})
	if err != nil {
		return nil, fmt.Errorf("db.ListAnswers: %w", err)
	}
	answers := make(map[string]dto.Answer, len(rows))
	for _, row := range rows {
		answers[row.QuestionHash] = dto.Answer{
			PYes: float64(row.PYes), PNo: float64(row.PNo),
			PNotStated: float64(row.PNotStated), Confidence: float64(row.Confidence),
		}
	}
	return answers, nil
}

// CompleteAnswerEffect writes the effect's answers and every surviving
// user's score, then marks it done, in one transaction.
func (db *DB) CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	effectID, err := parseUUID(effect.ID)
	if err != nil {
		return nil, err
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin complete answer effect: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.queries.WithTx(tx)

	rows, err := queries.CompleteAnswerEffect(ctx, pgsqlc.CompleteAnswerEffectParams{ID: effectID, Attempts: int32(effect.Attempts)})
	if err != nil {
		return nil, fmt.Errorf("db.CompleteAnswerEffect: %w", err)
	}
	if rows == 0 {
		return nil, tx.Commit(ctx)
	}

	jobID, err := parseUUID(effect.JobID)
	if err != nil {
		return nil, err
	}
	job, err := queries.GetJobForScoring(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("db.CompleteAnswerEffect: reload job: %w", err)
	}
	if job.ContentFingerprint.String != effect.Fingerprint {
		return nil, tx.Commit(ctx)
	}

	for hash, a := range answers {
		err := queries.InsertOptionAnswer(ctx, pgsqlc.InsertOptionAnswerParams{
			JobID: jobID, Fingerprint: effect.Fingerprint, QuestionHash: hash, Model: effect.Model,
			PYes: float32(a.PYes), PNo: float32(a.PNo), PNotStated: float32(a.PNotStated), Confidence: float32(a.Confidence),
		})
		if err != nil {
			return nil, fmt.Errorf("db.CompleteAnswerEffect: insert answer: %w", err)
		}
	}

	saved := make([]string, 0, len(scores))
	for _, s := range scores {
		if err := upsertJobScore(ctx, queries, s, effect.Fingerprint, effect.Model); err != nil {
			return nil, fmt.Errorf("db.CompleteAnswerEffect: %w", err)
		}
		saved = append(saved, s.UserID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit complete answer effect: %w", err)
	}
	return saved, nil
}

func upsertJobScore(ctx context.Context, queries *pgsqlc.Queries, s dto.JobScore, fingerprint, model string) error {
	jobID, err := parseUUID(s.JobID)
	if err != nil {
		return err
	}
	userID, err := parseUUID(s.UserID)
	if err != nil {
		return err
	}
	breakdown, err := json.Marshal(s.Rows)
	if err != nil {
		return fmt.Errorf("marshal breakdown: %w", err)
	}
	cost, err := toNumeric(s.Cost)
	if err != nil {
		return fmt.Errorf("cost: %w", err)
	}
	return queries.UpsertJobScore(ctx, pgsqlc.UpsertJobScoreParams{
		JobID: jobID, UserID: userID, Score: int32(s.Score), Breakdown: breakdown, Cost: cost,
		Fingerprint: fingerprint, Model: model,
	})
}

func (db *DB) ListScoringInputs(ctx context.Context, userID string) ([]providers.ScoringInput, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, providers.ErrInvalidID
	}
	jobRows, err := db.queries.ListScoringInputJobs(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.ListScoringInputs: %w", err)
	}
	answerRows, err := db.queries.ListScoringAnswersForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.ListScoringInputs: %w", err)
	}
	answersByJob := make(map[string]map[string]dto.Answer, len(jobRows))
	for _, a := range answerRows {
		jobID := a.JobID.String()
		if answersByJob[jobID] == nil {
			answersByJob[jobID] = make(map[string]dto.Answer)
		}
		answersByJob[jobID][a.QuestionHash] = dto.Answer{
			PYes: float64(a.PYes), PNo: float64(a.PNo), PNotStated: float64(a.PNotStated), Confidence: float64(a.Confidence),
		}
	}

	inputs := make([]providers.ScoringInput, len(jobRows))
	for i, row := range jobRows {
		job := dto.Job{
			ID: row.ID.String(), Title: row.Title, Location: row.Location, URL: row.Url,
			CompanySlug: row.CompanySlug, Source: row.Source, UpdatedAt: row.UpdatedAt.Time,
			ScrapedAt: row.ScrapedAt.Time, Description: row.Description, SalaryRaw: row.SalaryRaw,
			WorkArrangement: row.WorkArrangement, CompanyID: uuidString(row.CompanyID),
			BoardID: uuidString(row.PrimaryBoardID), ProviderPostingID: row.ProviderPostingID.String,
			ContentFingerprint: row.ContentFingerprint.String,
		}
		inputs[i] = providers.ScoringInput{Job: job, Answers: answersByJob[job.ID]}
	}
	return inputs, nil
}

func (db *DB) SaveScores(ctx context.Context, scores []dto.JobScore) error {
	for _, s := range scores {
		jobID, err := parseUUID(s.JobID)
		if err != nil {
			return err
		}
		userID, err := parseUUID(s.UserID)
		if err != nil {
			return err
		}
		breakdown, err := json.Marshal(s.Rows)
		if err != nil {
			return fmt.Errorf("db.SaveScores: marshal breakdown: %w", err)
		}
		if err := db.queries.UpdateJobScoreBreakdown(ctx, pgsqlc.UpdateJobScoreBreakdownParams{
			Score: int32(s.Score), Breakdown: breakdown, JobID: jobID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("db.SaveScores: %w", err)
		}
	}
	return nil
}

func (db *DB) GetScoringStatus(ctx context.Context, userID string) (dto.ScoringStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.ScoringStatus{}, providers.ErrInvalidID
	}
	pending, err := db.queries.GetScoringStatus(ctx, uid)
	if err != nil {
		return dto.ScoringStatus{}, fmt.Errorf("db.GetScoringStatus: %w", err)
	}
	return dto.ScoringStatus{Pending: pending}, nil
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return id.String()
}

func toNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.ScanScientific(strconv.FormatFloat(f, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}
