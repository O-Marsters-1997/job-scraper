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

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store/sqlc"
)

// ScoringInput is one job and its cached answers, keyed by question hash,
// ready for Recompute.
type ScoringInput struct {
	Job       dto.Job
	Answers   map[string]dto.Answer
	Favourite bool
	// Corrections maps option id to the user's "yes" or "no" for this job.
	Corrections map[string]string
}

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool)}
}

// JobsChanged drops jobIDs' stale cached answers and queues a fresh answer
// effect for each within tx (ADR 0011); firstDiscovery controls whether
// scoring's Run loop alerts once the effect completes.
func (s *Store) JobsChanged(ctx context.Context, tx pgx.Tx, jobIDs []string, firstDiscovery bool) error {
	ids, err := data.UUIDs(jobIDs)
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
		err = queries.QueueAnswerEffect(ctx, sqlc.QueueAnswerEffectParams{
			JobID: jobID, Fingerprint: job.ContentFingerprint.String, FirstDiscovery: firstDiscovery,
		})
		if err != nil {
			return fmt.Errorf("store.JobsChanged: queue answer effect: %w", err)
		}
	}
	return nil
}

// JobsClosed drops jobIDs' cached answers within tx (ADR 0011).
func (s *Store) JobsClosed(ctx context.Context, tx pgx.Tx, jobIDs []string) error {
	ids, err := data.UUIDs(jobIDs)
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
	cid, err := data.UUID(companyID)
	if err != nil {
		return err
	}
	if err := s.queries.WithTx(tx).QueueTrackingScores(ctx, cid); err != nil {
		return fmt.Errorf("store.CompanyTracked: %w", err)
	}
	return nil
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
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.SearchConfig{}, err
	}
	row, err := s.queries.GetSearchConfig(ctx, uid)
	if err != nil {
		return dto.SearchConfig{}, data.QueryErr("GetSearchConfig", err)
	}
	cfg, err := toSearchConfigDTO(row)
	if err != nil {
		return dto.SearchConfig{}, fmt.Errorf("store.GetSearchConfig: %w", err)
	}
	return cfg, nil
}

func (s *Store) ListIncludeFilterConfigs(ctx context.Context) ([]dto.SearchConfig, error) {
	rows, err := s.queries.ListIncludeFilterConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("store.ListIncludeFilterConfigs: %w", err)
	}
	configs := make([]dto.SearchConfig, 0, len(rows))
	for _, row := range rows {
		cfg, err := toSearchConfigDTO(row)
		if err != nil {
			return nil, fmt.Errorf("store.ListIncludeFilterConfigs: %w", err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

func (s *Store) UpsertSearchConfig(ctx context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error) {
	uid, err := data.UUID(cfg.UserID)
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
		RequiredLocations:     nonNilStrings(cfg.RequiredLocations),
		RequiredTitleKeywords: nonNilStrings(cfg.RequiredTitleKeywords),
		NotifyThreshold:       int32(cfg.NotifyThreshold),
		Preferences:           prefs,
		MaxJobAgeDays:         int32(cfg.MaxJobAgeDays),
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

func (s *Store) AddExcludedCompany(ctx context.Context, userID, name string) (bool, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return false, err
	}
	_, err = s.queries.AddExcludedCompany(ctx, sqlc.AddExcludedCompanyParams{UserID: uid, Name: name})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store.AddExcludedCompany: %w", err)
	}
	return true, nil
}

func (s *Store) RemoveExcludedCompany(ctx context.Context, userID, name string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.RemoveExcludedCompany(ctx, sqlc.RemoveExcludedCompanyParams{UserID: uid, Name: name}); err != nil {
		return fmt.Errorf("store.RemoveExcludedCompany: %w", err)
	}
	return nil
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
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		queries := s.queries.WithTx(tx)
		err := queries.InsertScoringOption(ctx, sqlc.InsertScoringOptionParams{
			ID: id, Dimension: sqlc.ScoringDimension(dimension), Label: label, Question: question,
		})
		if err != nil {
			return fmt.Errorf("store.AddScoringOption: %w", err)
		}
		if err := queries.QueueOptionBackfill(ctx); err != nil {
			return fmt.Errorf("store.AddScoringOption: queue backfill: %w", err)
		}
		return nil
	})
}

// RewordScoringOption changes an option's question text, which changes its
// question hash, and queues the same backfill as AddScoringOption.
func (s *Store) RewordScoringOption(ctx context.Context, id, question string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
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
		return nil
	})
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
	uid, err := data.UUID(userID)
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
	if err != nil {
		return dto.AnswerEffect{}, data.QueryErr("ClaimAnswerEffect", err)
	}
	return dto.AnswerEffect{
		ID: row.ID.String(), JobID: row.JobID.String(), Fingerprint: row.Fingerprint,
		Model: row.Model, Attempts: int(row.Attempts), FirstDiscovery: row.FirstDiscovery,
	}, nil
}

func (s *Store) FailAnswerEffect(ctx context.Context, id string, attempts int, failure dto.ScoringFailure) error {
	effectID, err := data.UUID(id)
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
	jid, err := data.UUID(jobID)
	if err != nil {
		return dto.Job{}, data.ErrNotFound
	}
	row, err := s.queries.GetJobForScoring(ctx, jid)
	if err != nil {
		return dto.Job{}, data.QueryErr("GetJobForScoring", err)
	}
	return toJobDTO(row), nil
}

// GetCompanyProfile returns the most recently fetched profile of companyID,
// or data.ErrNotFound when it has none.
func (s *Store) GetCompanyProfile(ctx context.Context, companyID string) (dto.CompanyProfile, error) {
	cid, err := data.UUID(companyID)
	if err != nil {
		return dto.CompanyProfile{}, data.ErrNotFound
	}
	raw, err := s.queries.GetCompanyProfile(ctx, cid)
	if err != nil {
		return dto.CompanyProfile{}, data.QueryErr("GetCompanyProfile", err)
	}
	var profile dto.CompanyProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return dto.CompanyProfile{}, fmt.Errorf("store.GetCompanyProfile: unmarshal: %w", err)
	}
	return profile, nil
}

func (s *Store) ListInterestedConfigs(ctx context.Context, jobID string) ([]dto.SearchConfig, error) {
	jid, err := data.UUID(jobID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListInterestedConfigs(ctx, jid)
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
			RequiredLocations:     row.RequiredLocations,
			RequiredTitleKeywords: row.RequiredTitleKeywords,
			NotifyThreshold:       int(row.NotifyThreshold),
			MaxJobAgeDays:         int(row.MaxJobAgeDays),
			CompanyIsNew:          row.CompanyIsNew,
			CompanyIsFavourite:    row.CompanyIsFavourite,
			Preferences:           prefs,
		}
	}
	return configs, nil
}

func (s *Store) ListAnswers(ctx context.Context, jobID, fingerprint, model string) (map[string]dto.Answer, error) {
	jid, err := data.UUID(jobID)
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
	jid, err := data.UUID(jobID)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
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
		return nil
	})
}

// CompleteAnswerEffect writes the effect's answers and every surviving
// user's score, then marks it done, in one transaction.
func (s *Store) CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	effectID, err := data.UUID(effect.ID)
	if err != nil {
		return nil, err
	}
	var saved []string
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		saved, err = s.completeAnswerEffect(ctx, s.queries.WithTx(tx), effect, effectID, answers, scores)
		return err
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *Store) completeAnswerEffect(ctx context.Context, queries *sqlc.Queries, effect dto.AnswerEffect, effectID pgtype.UUID, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	rows, err := queries.CompleteAnswerEffect(ctx, sqlc.CompleteAnswerEffectParams{ID: effectID, Attempts: int32(effect.Attempts)})
	if err != nil {
		return nil, fmt.Errorf("store.CompleteAnswerEffect: %w", err)
	}
	if rows == 0 {
		return nil, nil
	}

	jobID, err := data.UUID(effect.JobID)
	if err != nil {
		return nil, err
	}
	job, err := queries.GetJobForScoring(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("store.CompleteAnswerEffect: reload job: %w", err)
	}
	if job.ContentFingerprint.String != effect.Fingerprint {
		return nil, nil
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
	return saved, nil
}

func upsertJobScore(ctx context.Context, queries *sqlc.Queries, sc dto.JobScore, fingerprint, model string) error {
	jobID, err := data.UUID(sc.JobID)
	if err != nil {
		return err
	}
	userID, err := data.UUID(sc.UserID)
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
		JobID: jobID, UserID: userID, Score: int32(sc.Score), Band: sc.Band, Breakdown: breakdown, Cost: cost,
		Fingerprint: fingerprint, Model: model,
	})
}

func (s *Store) ListScoringInputs(ctx context.Context, userID, model string) ([]ScoringInput, error) {
	return listScoringInputs(ctx, s.queries, userID, model, pgtype.UUID{})
}

// ListCompanyScoringInputs is ListScoringInputs narrowed to userID's scored,
// open Jobs at companyID, read within tx.
func (s *Store) ListCompanyScoringInputs(ctx context.Context, tx pgx.Tx, userID, companyID, model string) ([]ScoringInput, error) {
	cid, err := data.UUID(companyID)
	if err != nil {
		return nil, err
	}
	return listScoringInputs(ctx, s.queries.WithTx(tx), userID, model, cid)
}

// IsJobCompanyFavourite reports whether userID has starred jobID's Company.
func (s *Store) IsJobCompanyFavourite(ctx context.Context, userID, jobID string) (bool, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return false, err
	}
	jid, err := data.UUID(jobID)
	if err != nil {
		return false, err
	}
	favourite, err := s.queries.IsJobCompanyFavourite(ctx, sqlc.IsJobCompanyFavouriteParams{JobID: jid, UserID: uid})
	if err != nil {
		return false, fmt.Errorf("store.IsJobCompanyFavourite: %w", err)
	}
	return favourite, nil
}

func (s *Store) SaveScoresTx(ctx context.Context, tx pgx.Tx, scores []dto.JobScore) error {
	return saveScores(ctx, s.queries.WithTx(tx), scores)
}

func listScoringInputs(ctx context.Context, queries *sqlc.Queries, userID, model string, companyID pgtype.UUID) ([]ScoringInput, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	jobRows, err := queries.ListScoringInputJobs(ctx, sqlc.ListScoringInputJobsParams{UserID: uid, CompanyID: companyID})
	if err != nil {
		return nil, fmt.Errorf("store.ListScoringInputs: %w", err)
	}
	answerRows, err := queries.ListScoringAnswersForUser(ctx, sqlc.ListScoringAnswersForUserParams{UserID: uid, Model: model})
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

	correctionRows, err := queries.ListAnswerCorrectionsForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListScoringInputs: corrections: %w", err)
	}
	correctionsByJob := make(map[string]map[string]string)
	for _, c := range correctionRows {
		jobID := c.JobID.String()
		if correctionsByJob[jobID] == nil {
			correctionsByJob[jobID] = make(map[string]string)
		}
		correctionsByJob[jobID][c.OptionID] = c.Value
	}

	inputs := make([]ScoringInput, len(jobRows))
	for i, row := range jobRows {
		job := toJobDTO(sqlc.GetJobForScoringRow{
			ID: row.ID, Title: row.Title, Location: row.Location, Url: row.Url, CompanySlug: row.CompanySlug,
			Source: row.Source, UpdatedAt: row.UpdatedAt, ScrapedAt: row.ScrapedAt, Description: row.Description,
			SalaryRaw: row.SalaryRaw, WorkArrangement: row.WorkArrangement, CompanyID: row.CompanyID,
			PrimaryBoardID: row.PrimaryBoardID, ProviderPostingID: row.ProviderPostingID,
			ContentFingerprint: row.ContentFingerprint,
		})
		inputs[i] = ScoringInput{Job: job, Answers: answersByJob[job.ID], Corrections: correctionsByJob[job.ID], Favourite: row.IsFavourite}
	}
	return inputs, nil
}

func (s *Store) SaveScores(ctx context.Context, scores []dto.JobScore) error {
	return saveScores(ctx, s.queries, scores)
}

func saveScores(ctx context.Context, queries *sqlc.Queries, scores []dto.JobScore) error {
	for _, sc := range scores {
		jobID, err := data.UUID(sc.JobID)
		if err != nil {
			return err
		}
		userID, err := data.UUID(sc.UserID)
		if err != nil {
			return err
		}
		breakdown, err := json.Marshal(sc.Rows)
		if err != nil {
			return fmt.Errorf("store.SaveScores: marshal breakdown: %w", err)
		}
		if err := queries.UpdateJobScoreBreakdown(ctx, sqlc.UpdateJobScoreBreakdownParams{
			Score: int32(sc.Score), Band: sc.Band, Breakdown: breakdown, JobID: jobID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("store.SaveScores: %w", err)
		}
	}
	return nil
}

func (s *Store) GetScoringStatus(ctx context.Context, userID string) (dto.ScoringStatus, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.ScoringStatus{}, err
	}
	pending, err := s.queries.GetScoringStatus(ctx, uid)
	if err != nil {
		return dto.ScoringStatus{}, fmt.Errorf("store.GetScoringStatus: %w", err)
	}
	return dto.ScoringStatus{Pending: pending}, nil
}

func countsBy[R any](rows []R, kv func(R) (string, int64)) map[string]int64 {
	counts := make(map[string]int64, len(rows))
	for _, r := range rows {
		k, v := kv(r)
		counts[k] = v
	}
	return counts
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
	disabled, err := s.queries.DisabledSourceTargets(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState disabled source targets: %w", err)
	}
	disabledBySource := countsBy(disabled, func(r sqlc.DisabledSourceTargetsRow) (string, int64) { return r.Source, r.Disabled })
	unique, err := s.queries.UniqueRelevantJobs(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState unique relevant jobs: %w", err)
	}
	uniqueBySource := countsBy(unique, func(r sqlc.UniqueRelevantJobsRow) (string, int64) { return r.Source, r.Jobs })
	boards, err := s.queries.DiscoveryBoards(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState discovery boards: %w", err)
	}
	boardsByVia := countsBy(boards, func(r sqlc.DiscoveryBoardsRow) (string, int64) { return r.Via, r.Boards })
	relevant, err := s.queries.DiscoveryRelevantJobs(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState discovery relevant jobs: %w", err)
	}
	relevantByVia := countsBy(relevant, func(r sqlc.DiscoveryRelevantJobsRow) (string, int64) { return r.Via, r.Jobs })
	admitted, err := s.queries.HarvestAdmitted(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState harvest admitted: %w", err)
	}
	admittedByHarvester := countsBy(admitted, func(r sqlc.HarvestAdmittedRow) (string, int64) { return r.Harvester, r.Companies })
	emptied, err := s.queries.EmptiedBoards(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState emptied boards: %w", err)
	}
	emptiedBySource := countsBy(emptied, func(r sqlc.EmptiedBoardsRow) (string, int64) { return r.Source, r.Boards })
	underparsed, err := s.queries.UnderparsedBoards(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState underparsed boards: %w", err)
	}
	underparsedBySource := countsBy(underparsed, func(r sqlc.UnderparsedBoardsRow) (string, int64) { return r.Source, r.Boards })
	completeness, err := s.queries.FieldCompleteness(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("store.OpsState field completeness: %w", err)
	}
	completenessBySource := make(map[string]map[string]float64)
	for _, r := range completeness {
		if completenessBySource[r.Source] == nil {
			completenessBySource[r.Source] = make(map[string]float64)
		}
		completenessBySource[r.Source][r.Field] = r.Share
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
		DisabledSourceTargets:  disabledBySource,
		HarvestAge:             harvestAge,
		UniqueRelevantJobs:     uniqueBySource,
		DiscoveryBoards:        boardsByVia,
		DiscoveryRelevantJobs:  relevantByVia,
		HarvestAdmitted:        admittedByHarvester,
		EmptiedBoards:          emptiedBySource,
		UnderparsedBoards:      underparsedBySource,
		FieldCompleteness:      completenessBySource,
	}, nil
}

// ListCompanyAnswers returns, per company, one answer map (keyed by question
// hash) for each of its open jobs, from the cache only.
func (s *Store) ListCompanyAnswers(ctx context.Context, companyIDs []string, model string) (map[string][]map[string]dto.Answer, error) {
	ids, err := data.UUIDs(companyIDs)
	if err != nil {
		return nil, err
	}
	jobs, err := s.queries.ListOpenCompanyJobs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("store.ListCompanyAnswers: %w", err)
	}
	rows, err := s.queries.ListCompanyJobAnswers(ctx, sqlc.ListCompanyJobAnswersParams{CompanyIds: ids, Model: model})
	if err != nil {
		return nil, fmt.Errorf("store.ListCompanyAnswers: %w", err)
	}
	byJob := make(map[string]map[string]dto.Answer, len(jobs))
	for _, a := range rows {
		jobID := a.JobID.String()
		if byJob[jobID] == nil {
			byJob[jobID] = make(map[string]dto.Answer)
		}
		byJob[jobID][a.QuestionHash] = dto.Answer{
			PYes: float64(a.PYes), PNo: float64(a.PNo), PNotStated: float64(a.PNotStated), Confidence: float64(a.Confidence),
		}
	}
	out := make(map[string][]map[string]dto.Answer)
	for _, j := range jobs {
		companyID := j.CompanyID.String()
		out[companyID] = append(out[companyID], byJob[j.ID.String()])
	}
	return out, nil
}

// InsertScoreFeedback appends one entry to userID's Score Feedback log.
func (s *Store) InsertScoreFeedback(ctx context.Context, userID string, f dto.ScoreFeedback) (dto.ScoreFeedback, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.ScoreFeedback{}, err
	}
	picks, err := json.Marshal(f.Picks)
	if err != nil {
		return dto.ScoreFeedback{}, fmt.Errorf("store.InsertScoreFeedback: %w", err)
	}
	snapshot, err := json.Marshal(f.Snapshot)
	if err != nil {
		return dto.ScoreFeedback{}, fmt.Errorf("store.InsertScoreFeedback: %w", err)
	}
	var jobID pgtype.UUID
	if f.JobID != nil {
		if jobID, err = data.UUID(*f.JobID); err != nil {
			return dto.ScoreFeedback{}, err
		}
	}
	var direction pgtype.Text
	if f.Direction != nil {
		direction = data.Text(*f.Direction)
	}
	row, err := s.queries.InsertScoreFeedback(ctx, sqlc.InsertScoreFeedbackParams{
		UserID: uid, JobID: jobID, Kind: f.Kind, Direction: direction, Reason: f.Reason,
		Picks: picks, Model: f.Model, Snapshot: snapshot,
	})
	if err != nil {
		return dto.ScoreFeedback{}, fmt.Errorf("store.InsertScoreFeedback: %w", err)
	}
	out, err := toScoreFeedbackDTO(row)
	if err != nil {
		return dto.ScoreFeedback{}, fmt.Errorf("store.InsertScoreFeedback: %w", err)
	}
	return out, nil
}

// ListScoreFeedback returns userID's entries matching f, newest first,
// skipping offset and returning at most limit. Each entry carries its drift
// from the user's current Picks and f.Model.
func (s *Store) ListScoreFeedback(ctx context.Context, userID string, f dto.ScoreFeedbackFilter, limit, offset int) ([]dto.ScoreFeedback, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListScoreFeedback(ctx, sqlc.ListScoreFeedbackParams{
		UserID: uid, Kind: f.Kind, Model: f.Model, IncludeOutdated: f.IncludeOutdated,
		RowLimit: int32(limit), RowOffset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("store.ListScoreFeedback: %w", err)
	}
	out := make([]dto.ScoreFeedback, len(rows))
	for i, row := range rows {
		if out[i], err = toListedScoreFeedbackDTO(row); err != nil {
			return nil, fmt.Errorf("store.ListScoreFeedback: %w", err)
		}
	}
	return out, nil
}

// CountScoreFeedback counts userID's current and outdated entries of f.Kind
// (all kinds when empty) against f.Model.
func (s *Store) CountScoreFeedback(ctx context.Context, userID string, f dto.ScoreFeedbackFilter) (current, outdated int, err error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return 0, 0, err
	}
	row, err := s.queries.CountScoreFeedback(ctx, sqlc.CountScoreFeedbackParams{UserID: uid, Kind: f.Kind, Model: f.Model})
	if err != nil {
		return 0, 0, fmt.Errorf("store.CountScoreFeedback: %w", err)
	}
	return int(row.Current), int(row.Outdated), nil
}

// DeleteScoreFeedback removes one of userID's entries; data.ErrNotFound when
// id is unknown or belongs to someone else.
func (s *Store) DeleteScoreFeedback(ctx context.Context, userID, id string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	fid, err := data.UUID(id)
	if err != nil {
		return data.ErrNotFound
	}
	n, err := s.queries.DeleteScoreFeedback(ctx, sqlc.DeleteScoreFeedbackParams{ID: fid, UserID: uid})
	if err != nil {
		return fmt.Errorf("store.DeleteScoreFeedback: %w", err)
	}
	if n == 0 {
		return data.ErrNotFound
	}
	return nil
}

// ClearScoreFeedback hard-deletes userID's whole log and returns the count.
func (s *Store) ClearScoreFeedback(ctx context.Context, userID string) (int64, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return 0, err
	}
	n, err := s.queries.ClearScoreFeedback(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("store.ClearScoreFeedback: %w", err)
	}
	return n, nil
}

// GetJobScoreForFeedback returns userID's stored score for jobID, or
// data.ErrNotFound when the Job has none.
func (s *Store) GetJobScoreForFeedback(ctx context.Context, userID, jobID string) (dto.JobScoreEvidence, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.JobScoreEvidence{}, err
	}
	jid, err := data.UUID(jobID)
	if err != nil {
		return dto.JobScoreEvidence{}, err
	}
	row, err := s.queries.GetJobScoreForFeedback(ctx, sqlc.GetJobScoreForFeedbackParams{UserID: uid, JobID: jid})
	if err != nil {
		return dto.JobScoreEvidence{}, data.QueryErr("GetJobScoreForFeedback", err)
	}
	rows := []dto.ScoreRow{}
	if err := json.Unmarshal(row.Breakdown, &rows); err != nil {
		return dto.JobScoreEvidence{}, fmt.Errorf("store.GetJobScoreForFeedback: breakdown: %w", err)
	}
	return dto.JobScoreEvidence{Score: int(row.Score), Band: row.Band, Breakdown: rows, Fingerprint: row.ScoreFingerprint, Model: row.ScoreModel}, nil
}

// ListJobScoresForCollection returns userID's stored score for each of jobIDs
// that exists, in no particular order. A Job with no score has a nil Score.
func (s *Store) ListJobScoresForCollection(ctx context.Context, userID string, jobIDs []string) ([]dto.CollectionJobScore, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	ids, err := data.UUIDs(jobIDs)
	if err != nil {
		return nil, apperr.Invalid("jobIds must be valid ids")
	}
	rows, err := s.queries.ListJobScoresForCollection(ctx, sqlc.ListJobScoresForCollectionParams{UserID: uid, JobIds: ids})
	if err != nil {
		return nil, fmt.Errorf("store.ListJobScoresForCollection: %w", err)
	}
	out := make([]dto.CollectionJobScore, len(rows))
	for i, row := range rows {
		breakdown := []dto.ScoreRow{}
		if err := json.Unmarshal(row.Breakdown, &breakdown); err != nil {
			return nil, fmt.Errorf("store.ListJobScoresForCollection: breakdown: %w", err)
		}
		out[i] = dto.CollectionJobScore{JobID: row.ID.String(), Title: row.Title, Company: row.CompanySlug, Breakdown: breakdown}
		if row.Score.Valid {
			score := int(row.Score.Int32)
			out[i].Score = &score
		}
	}
	return out, nil
}

// UpsertGrade writes userID's Grade for g.JobID, replacing any earlier one.
func (s *Store) UpsertGrade(ctx context.Context, userID string, g dto.Grade) (dto.Grade, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Grade{}, err
	}
	jid, err := data.UUID(g.JobID)
	if err != nil {
		return dto.Grade{}, data.ErrNotFound
	}
	params := sqlc.UpsertGradeParams{UserID: uid, JobID: jid, Grade: g.Grade, Reasons: nonNilStrings(g.Reasons)}
	if g.ScoreAtGrade != nil {
		params.ScoreAtGrade = pgtype.Int4{Int32: int32(*g.ScoreAtGrade), Valid: true}
	}
	if g.ScoreModel != "" {
		params.ScoreModel = data.Text(g.ScoreModel)
	}
	row, err := s.queries.UpsertGrade(ctx, params)
	if err != nil {
		return dto.Grade{}, fmt.Errorf("store.UpsertGrade: %w", err)
	}
	return toGradeDTO(row), nil
}

// GetGrade returns userID's Grade for jobID, or data.ErrNotFound.
func (s *Store) GetGrade(ctx context.Context, userID, jobID string) (dto.Grade, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Grade{}, err
	}
	jid, err := data.UUID(jobID)
	if err != nil {
		return dto.Grade{}, data.ErrNotFound
	}
	row, err := s.queries.GetGrade(ctx, sqlc.GetGradeParams{UserID: uid, JobID: jid})
	if err != nil {
		return dto.Grade{}, data.QueryErr("GetGrade", err)
	}
	return toGradeDTO(row), nil
}

// DeleteGrade removes userID's Grade for jobID; an ungraded job is a no-op.
func (s *Store) DeleteGrade(ctx context.Context, userID, jobID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	jid, err := data.UUID(jobID)
	if err != nil {
		return nil
	}
	if _, err := s.queries.DeleteGrade(ctx, sqlc.DeleteGradeParams{UserID: uid, JobID: jid}); err != nil {
		return fmt.Errorf("store.DeleteGrade: %w", err)
	}
	return nil
}

// ListGrades returns every Grade userID has given, newest first.
func (s *Store) ListGrades(ctx context.Context, userID string) ([]dto.Grade, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListGrades(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListGrades: %w", err)
	}
	out := make([]dto.Grade, len(rows))
	for i, row := range rows {
		out[i] = toGradeDTO(row)
	}
	return out, nil
}

// ListImpliedPositives returns userID's applications and kept tailored CVs
// as implied positive labels.
func (s *Store) ListImpliedPositives(ctx context.Context, userID string) ([]dto.ImpliedLabel, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListImpliedPositives(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListImpliedPositives: %w", err)
	}
	out := make([]dto.ImpliedLabel, len(rows))
	for i, row := range rows {
		out[i] = dto.ImpliedLabel{JobID: row.JobID.String(), Source: row.Source}
	}
	return out, nil
}

func (s *Store) SetAnswerCorrection(ctx context.Context, userID, jobID, optionID, value string) error {
	uid, jid, err := userAndJob(userID, jobID)
	if err != nil {
		return err
	}
	if err := s.queries.UpsertAnswerCorrection(ctx, sqlc.UpsertAnswerCorrectionParams{UserID: uid, JobID: jid, OptionID: optionID, Value: value}); err != nil {
		return fmt.Errorf("store.SetAnswerCorrection: %w", err)
	}
	return nil
}

func (s *Store) DeleteAnswerCorrection(ctx context.Context, userID, jobID, optionID string) error {
	uid, jid, err := userAndJob(userID, jobID)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteAnswerCorrection(ctx, sqlc.DeleteAnswerCorrectionParams{UserID: uid, JobID: jid, OptionID: optionID}); err != nil {
		return fmt.Errorf("store.DeleteAnswerCorrection: %w", err)
	}
	return nil
}

// ListJobCorrections returns every user's Corrections for jobID, keyed by
// user id then option id.
func (s *Store) ListJobCorrections(ctx context.Context, jobID string) (map[string]map[string]string, error) {
	jid, err := data.UUID(jobID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListAnswerCorrectionsForJob(ctx, jid)
	if err != nil {
		return nil, fmt.Errorf("store.ListJobCorrections: %w", err)
	}
	out := make(map[string]map[string]string)
	for _, r := range rows {
		userID := r.UserID.String()
		if out[userID] == nil {
			out[userID] = make(map[string]string)
		}
		out[userID][r.OptionID] = r.Value
	}
	return out, nil
}

func userAndJob(userID, jobID string) (pgtype.UUID, pgtype.UUID, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	jid, err := data.UUID(jobID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return uid, jid, nil
}
