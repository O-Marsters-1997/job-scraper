// Package store is the scoring context's Postgres store: search config,
// scoring options, option answers, job scores and the answer-effect outbox.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store/sqlc"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

// ScoringInput is one job and its cached answers, keyed by question hash,
// ready for Recompute.
type ScoringInput struct {
	Job     dto.Job
	Answers map[string]dto.Answer
}

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool)}
}

func parseUUIDs(ids []string) ([]pgtype.UUID, error) {
	out := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		uid, err := parseUUID(id)
		if err != nil {
			return nil, err
		}
		out[i] = uid
	}
	return out, nil
}

// JobsChanged drops jobIDs' stale cached answers and queues a fresh answer
// effect for each within tx (ADR 0011); firstDiscovery controls whether
// scoring's Run loop alerts once the effect completes.
func (s *Store) JobsChanged(ctx context.Context, tx pgx.Tx, jobIDs []string, firstDiscovery bool) error {
	ids, err := parseUUIDs(jobIDs)
	if err != nil {
		return err
	}
	queries := s.queries.WithTx(tx)
	if err := queries.DropStaleAnswers(ctx, ids); err != nil {
		return fmt.Errorf("store.JobsChanged: drop stale answers: %w", err)
	}
	for _, jobID := range ids {
		job, err := queries.GetJobForScoring(ctx, jobID)
		if err != nil {
			return fmt.Errorf("store.JobsChanged: load job: %w", err)
		}
		role, _ := sourcespec.SourceRole(job.Source)
		err = queries.QueueAnswerEffect(ctx, sqlc.QueueAnswerEffectParams{
			JobID: jobID, Fingerprint: job.ContentFingerprint.String,
			FirstDiscovery: firstDiscovery, Discovery: role == sourcespec.RoleDiscovery,
		})
		if err != nil {
			return fmt.Errorf("store.JobsChanged: queue answer effect: %w", err)
		}
	}
	return nil
}

// JobsClosed drops jobIDs' cached answers within tx (ADR 0011).
func (s *Store) JobsClosed(ctx context.Context, tx pgx.Tx, jobIDs []string) error {
	ids, err := parseUUIDs(jobIDs)
	if err != nil {
		return err
	}
	if err := s.queries.WithTx(tx).DeleteAnswersForJobs(ctx, ids); err != nil {
		return fmt.Errorf("store.JobsClosed: %w", err)
	}
	return nil
}

// CompanyTracked queues an answer effect for companyID's open, fingerprinted
// Jobs within tx, with first_discovery left false so scoring's Run loop
// doesn't alert (ADR 0011).
func (s *Store) CompanyTracked(ctx context.Context, tx pgx.Tx, userID, companyID string) error {
	cid, err := parseUUID(companyID)
	if err != nil {
		return err
	}
	if err := s.queries.WithTx(tx).QueueTrackingScores(ctx, cid); err != nil {
		return fmt.Errorf("store.CompanyTracked: %w", err)
	}
	return nil
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}

func toNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.ScanScientific(strconv.FormatFloat(f, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (s *Store) GetSearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SearchConfig{}, err
	}
	row, err := s.queries.GetSearchConfig(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.SearchConfig{}, data.ErrNotFound
	}
	if err != nil {
		return dto.SearchConfig{}, fmt.Errorf("store.GetSearchConfig: %w", err)
	}
	cfg, err := toSearchConfigDTO(row)
	if err != nil {
		return dto.SearchConfig{}, fmt.Errorf("store.GetSearchConfig: %w", err)
	}
	return cfg, nil
}

func (s *Store) UpsertSearchConfig(ctx context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error) {
	uid, err := parseUUID(cfg.UserID)
	if err != nil {
		return dto.SearchConfig{}, err
	}
	prefs, err := json.Marshal(cfg.Preferences)
	if err != nil {
		return dto.SearchConfig{}, fmt.Errorf("store.UpsertSearchConfig: marshal preferences: %w", err)
	}
	row, err := s.queries.UpsertSearchConfig(ctx, sqlc.UpsertSearchConfigParams{
		UserID:                uid,
		ExcludedTitleKeywords: nonNilStrings(cfg.ExcludedTitleKeywords),
		ExcludedCompanies:     nonNilStrings(cfg.ExcludedCompanies),
		ExcludedLocations:     nonNilStrings(cfg.ExcludedLocations),
		NotifyThreshold:       int32(cfg.NotifyThreshold),
		Preferences:           prefs,
	})
	if err != nil {
		return dto.SearchConfig{}, fmt.Errorf("store.UpsertSearchConfig: %w", err)
	}
	updated, err := toSearchConfigDTO(row)
	if err != nil {
		return dto.SearchConfig{}, fmt.Errorf("store.UpsertSearchConfig: %w", err)
	}
	return updated, nil
}

func (s *Store) ListScoringOptions(ctx context.Context) ([]dto.ScoringOption, error) {
	rows, err := s.queries.ListScoringOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("store.ListScoringOptions: %w", err)
	}
	options := make([]dto.ScoringOption, len(rows))
	for i, row := range rows {
		options[i] = toScoringOptionDTO(row)
	}
	return options, nil
}

// AddScoringOption inserts a bank option and queues an answer effect for
// every non-closed job that already has a job_scores row, so the new
// question reaches jobs already scored.
func (s *Store) AddScoringOption(ctx context.Context, id, dimension, label, question string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add scoring option: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	err = queries.InsertScoringOption(ctx, sqlc.InsertScoringOptionParams{
		ID: id, Dimension: sqlc.ScoringDimension(dimension), Label: label, Question: question,
	})
	if err != nil {
		return fmt.Errorf("store.AddScoringOption: %w", err)
	}
	if err := queries.QueueOptionBackfill(ctx); err != nil {
		return fmt.Errorf("store.AddScoringOption: queue backfill: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit add scoring option: %w", err)
	}
	return nil
}

// RewordScoringOption changes an option's question text, which changes its
// question hash, and queues the same backfill as AddScoringOption.
func (s *Store) RewordScoringOption(ctx context.Context, id, question string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reword scoring option: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	rows, err := queries.RewordScoringOption(ctx, sqlc.RewordScoringOptionParams{ID: id, Question: question})
	if err != nil {
		return fmt.Errorf("store.RewordScoringOption: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("store.RewordScoringOption: option %q: %w", id, data.ErrNotFound)
	}
	if err := queries.QueueOptionBackfill(ctx); err != nil {
		return fmt.Errorf("store.RewordScoringOption: queue backfill: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reword scoring option: %w", err)
	}
	return nil
}

// RetireScoringOption sets retired_at so the option is hidden from new
// pickers. Existing picks keep scoring from their cached answers.
func (s *Store) RetireScoringOption(ctx context.Context, id string) error {
	rows, err := s.queries.RetireScoringOption(ctx, id)
	if err != nil {
		return fmt.Errorf("store.RetireScoringOption: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("store.RetireScoringOption: option %q: %w", id, data.ErrNotFound)
	}
	return nil
}

// QueueMissingAnswers queues an answer effect, without alerting, for each of
// userID's open, scored, fingerprinted jobs missing any of hashes.
func (s *Store) QueueMissingAnswers(ctx context.Context, userID string, hashes []string, model string) (int64, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return 0, err
	}
	n, err := s.queries.QueueMissingAnswers(ctx, sqlc.QueueMissingAnswersParams{
		UserID: uid, QuestionHashes: hashes, Model: model,
	})
	if err != nil {
		return 0, fmt.Errorf("store.QueueMissingAnswers: %w", err)
	}
	return n, nil
}

func (s *Store) ClaimAnswerEffect(ctx context.Context) (dto.AnswerEffect, error) {
	row, err := s.queries.ClaimAnswerEffect(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.AnswerEffect{}, data.ErrNotFound
	}
	if err != nil {
		return dto.AnswerEffect{}, fmt.Errorf("store.ClaimAnswerEffect: %w", err)
	}
	return dto.AnswerEffect{
		ID: row.ID.String(), JobID: row.JobID.String(), Fingerprint: row.Fingerprint,
		Model: row.Model, Attempts: int(row.Attempts), FirstDiscovery: row.FirstDiscovery,
	}, nil
}

func (s *Store) FailAnswerEffect(ctx context.Context, id string, attempts int, failure dto.ScoringFailure) error {
	effectID, err := parseUUID(id)
	if err != nil {
		return err
	}
	var retryAfterSecs pgtype.Int4
	if failure.RetryAfter > 0 {
		retryAfterSecs = pgtype.Int4{Int32: int32(failure.RetryAfter.Seconds()), Valid: true}
	}
	err = s.queries.FailAnswerEffect(ctx, sqlc.FailAnswerEffectParams{
		ID: effectID, Attempts: int32(attempts), LastError: failure.Reason,
		Terminal: failure.Terminal, RetryAfterSecs: retryAfterSecs,
	})
	if err != nil {
		return fmt.Errorf("store.FailAnswerEffect: %w", err)
	}
	return nil
}

func (s *Store) GetJobForScoring(ctx context.Context, jobID string) (dto.Job, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return dto.Job{}, err
	}
	row, err := s.queries.GetJobForScoring(ctx, jid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Job{}, data.ErrNotFound
	}
	if err != nil {
		return dto.Job{}, fmt.Errorf("store.GetJobForScoring: %w", err)
	}
	return toJobDTO(row), nil
}

func (s *Store) ListInterestedConfigs(ctx context.Context, jobID string, discovery bool) ([]dto.SearchConfig, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListInterestedConfigs(ctx, sqlc.ListInterestedConfigsParams{JobID: jid, Discovery: discovery})
	if err != nil {
		return nil, fmt.Errorf("store.ListInterestedConfigs: %w", err)
	}
	configs := make([]dto.SearchConfig, len(rows))
	for i, row := range rows {
		var prefs dto.Preferences
		if err := json.Unmarshal(row.Preferences, &prefs); err != nil {
			return nil, fmt.Errorf("store.ListInterestedConfigs: unmarshal preferences: %w", err)
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

func (s *Store) ListAnswers(ctx context.Context, jobID, fingerprint, model string) (map[string]dto.Answer, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListOptionAnswers(ctx, sqlc.ListOptionAnswersParams{JobID: jid, Fingerprint: fingerprint, Model: model})
	if err != nil {
		return nil, fmt.Errorf("store.ListAnswers: %w", err)
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

// SaveAnswers caches answers for one job version, keeping any already stored.
func (s *Store) SaveAnswers(ctx context.Context, jobID, fingerprint, model string, answers map[string]dto.Answer) error {
	jid, err := parseUUID(jobID)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save answers: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	for hash, a := range answers {
		err := queries.InsertOptionAnswer(ctx, sqlc.InsertOptionAnswerParams{
			JobID: jid, Fingerprint: fingerprint, QuestionHash: hash, Model: model,
			PYes: float32(a.PYes), PNo: float32(a.PNo), PNotStated: float32(a.PNotStated), Confidence: float32(a.Confidence),
		})
		if err != nil {
			return fmt.Errorf("store.SaveAnswers: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit save answers: %w", err)
	}
	return nil
}

// CompleteAnswerEffect writes the effect's answers and every surviving
// user's score, then marks it done, in one transaction.
func (s *Store) CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	effectID, err := parseUUID(effect.ID)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin complete answer effect: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	rows, err := queries.CompleteAnswerEffect(ctx, sqlc.CompleteAnswerEffectParams{ID: effectID, Attempts: int32(effect.Attempts)})
	if err != nil {
		return nil, fmt.Errorf("store.CompleteAnswerEffect: %w", err)
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
		return nil, fmt.Errorf("store.CompleteAnswerEffect: reload job: %w", err)
	}
	if job.ContentFingerprint.String != effect.Fingerprint {
		return nil, tx.Commit(ctx)
	}

	for hash, a := range answers {
		err := queries.InsertOptionAnswer(ctx, sqlc.InsertOptionAnswerParams{
			JobID: jobID, Fingerprint: effect.Fingerprint, QuestionHash: hash, Model: effect.Model,
			PYes: float32(a.PYes), PNo: float32(a.PNo), PNotStated: float32(a.PNotStated), Confidence: float32(a.Confidence),
		})
		if err != nil {
			return nil, fmt.Errorf("store.CompleteAnswerEffect: insert answer: %w", err)
		}
	}

	saved := make([]string, 0, len(scores))
	for _, sc := range scores {
		if err := upsertJobScore(ctx, queries, sc, effect.Fingerprint, effect.Model); err != nil {
			return nil, fmt.Errorf("store.CompleteAnswerEffect: %w", err)
		}
		saved = append(saved, sc.UserID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit complete answer effect: %w", err)
	}
	return saved, nil
}

func upsertJobScore(ctx context.Context, queries *sqlc.Queries, sc dto.JobScore, fingerprint, model string) error {
	jobID, err := parseUUID(sc.JobID)
	if err != nil {
		return err
	}
	userID, err := parseUUID(sc.UserID)
	if err != nil {
		return err
	}
	breakdown, err := json.Marshal(sc.Rows)
	if err != nil {
		return fmt.Errorf("marshal breakdown: %w", err)
	}
	cost, err := toNumeric(sc.Cost)
	if err != nil {
		return fmt.Errorf("cost: %w", err)
	}
	return queries.UpsertJobScore(ctx, sqlc.UpsertJobScoreParams{
		JobID: jobID, UserID: userID, Score: int32(sc.Score), Breakdown: breakdown, Cost: cost,
		Fingerprint: fingerprint, Model: model,
	})
}

func (s *Store) ListScoringInputs(ctx context.Context, userID, model string) ([]ScoringInput, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	jobRows, err := s.queries.ListScoringInputJobs(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListScoringInputs: %w", err)
	}
	answerRows, err := s.queries.ListScoringAnswersForUser(ctx, sqlc.ListScoringAnswersForUserParams{UserID: uid, Model: model})
	if err != nil {
		return nil, fmt.Errorf("store.ListScoringInputs: %w", err)
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

	inputs := make([]ScoringInput, len(jobRows))
	for i, row := range jobRows {
		job := toJobDTO(sqlc.GetJobForScoringRow(row))
		inputs[i] = ScoringInput{Job: job, Answers: answersByJob[job.ID]}
	}
	return inputs, nil
}

func (s *Store) SaveScores(ctx context.Context, scores []dto.JobScore) error {
	for _, sc := range scores {
		jobID, err := parseUUID(sc.JobID)
		if err != nil {
			return err
		}
		userID, err := parseUUID(sc.UserID)
		if err != nil {
			return err
		}
		breakdown, err := json.Marshal(sc.Rows)
		if err != nil {
			return fmt.Errorf("store.SaveScores: marshal breakdown: %w", err)
		}
		if err := s.queries.UpdateJobScoreBreakdown(ctx, sqlc.UpdateJobScoreBreakdownParams{
			Score: int32(sc.Score), Breakdown: breakdown, JobID: jobID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("store.SaveScores: %w", err)
		}
	}
	return nil
}

func (s *Store) GetScoringStatus(ctx context.Context, userID string) (dto.ScoringStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.ScoringStatus{}, err
	}
	pending, err := s.queries.GetScoringStatus(ctx, uid)
	if err != nil {
		return dto.ScoringStatus{}, fmt.Errorf("store.GetScoringStatus: %w", err)
	}
	return dto.ScoringStatus{Pending: pending}, nil
}

func (s *Store) OpsState(ctx context.Context) (dto.OpsState, error) {
	row, err := s.queries.OpsState(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState: %w", err)
	}
	harvests, err := s.queries.HarvestRuns(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState harvest runs: %w", err)
	}
	var oldestPendingAge time.Duration
	if row.OldestPendingCreatedAt.Valid {
		oldestPendingAge = time.Since(row.OldestPendingCreatedAt.Time)
	}
	harvestAge := make(map[string]time.Duration, len(harvests))
	for _, h := range harvests {
		harvestAge[h.Harvester] = time.Since(h.LastSucceededAt.Time)
	}
	return dto.OpsState{
		OutboxPending:          row.OutboxPending,
		OutboxOldestPendingAge: oldestPendingAge,
		OutboxFailed:           row.OutboxFailed,
		BoardsOverdue:          row.BoardsOverdue,
		BoardsFailing:          row.BoardsFailing,
		SourceTargetsFailed:    row.SourceTargetsFailed,
		HarvestAge:             harvestAge,
	}, nil
}
