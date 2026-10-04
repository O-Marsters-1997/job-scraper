// Package scoring is the scoring context: search config, scoring options,
// option answers, job scores and the answer-effect outbox (ADR 0011).
package scoring

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/filter"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

const (
	maxConcurrentEffects    = 4
	defaultAnswerEffectTick = 2 * time.Second
)

// Store is the scoring context's full Postgres surface: what Service needs
// to drain the answer-effect queue, and what Module binds routes and
// tx-scoped ports to directly (ADR 0012).
type Store interface {
	ClaimAnswerEffect(ctx context.Context) (dto.AnswerEffect, error)
	FailAnswerEffect(ctx context.Context, id string, attempts int, failure dto.ScoringFailure) error
	GetJobForScoring(ctx context.Context, jobID string) (dto.Job, error)
	GetCompanyProfile(ctx context.Context, companyID string) (dto.CompanyProfile, error)
	ListInterestedConfigs(ctx context.Context, jobID string) ([]dto.SearchConfig, error)
	ListScoringOptions(ctx context.Context) ([]dto.ScoringOption, error)
	ListAnswers(ctx context.Context, jobID, fingerprint, model string) (map[string]dto.Answer, error)
	CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error)
	SaveAnswers(ctx context.Context, jobID, fingerprint, model string, answers map[string]dto.Answer) error
	GetSearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error)
	UpsertSearchConfig(ctx context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error)
	ListIncludeFilterConfigs(ctx context.Context) ([]dto.SearchConfig, error)
	AddExcludedCompany(ctx context.Context, userID, name string) (bool, error)
	RemoveExcludedCompany(ctx context.Context, userID, name string) error
	ListScoringInputs(ctx context.Context, userID, model string) ([]store.ScoringInput, error)
	ListCompanyScoringInputs(ctx context.Context, tx pgx.Tx, userID, companyID, model string) ([]store.ScoringInput, error)
	IsJobCompanyFavourite(ctx context.Context, userID, jobID string) (bool, error)
	SaveScoresTx(ctx context.Context, tx pgx.Tx, scores []dto.JobScore) error
	SetAnswerCorrection(ctx context.Context, userID, jobID, optionID, value string) error
	DeleteAnswerCorrection(ctx context.Context, userID, jobID, optionID string) error
	ListJobCorrections(ctx context.Context, jobID string) (map[string]map[string]string, error)
	SaveScores(ctx context.Context, scores []dto.JobScore) error
	QueueMissingAnswers(ctx context.Context, userID string, hashes []string, model string) (int64, error)
	ListCompanyAnswers(ctx context.Context, companyIDs []string, model string) (map[string][]map[string]dto.Answer, error)

	OpsState(ctx context.Context) (dto.OpsState, error)
	GetScoringStatus(ctx context.Context, userID string) (dto.ScoringStatus, error)
	JobsChanged(ctx context.Context, tx pgx.Tx, jobIDs []string, firstDiscovery bool) error
	JobsClosed(ctx context.Context, tx pgx.Tx, jobIDs []string) error
	CompanyTracked(ctx context.Context, tx pgx.Tx, userID, companyID string) error
	AddScoringOption(ctx context.Context, id, dimension, label, question string) error
	RewordScoringOption(ctx context.Context, id, question string) error
	RetireScoringOption(ctx context.Context, id string) error
	UpsertPushSubscription(ctx context.Context, userID string, sub dto.PushSubscriptionInput) error
	ListPushSubscriptions(ctx context.Context, userID string) ([]dto.PushSubscriptionInput, error)
	DeletePushSubscription(ctx context.Context, userID, endpoint string) error
	InsertScoreFeedback(ctx context.Context, userID string, entry dto.ScoreFeedback) (dto.ScoreFeedback, error)
	ListScoreFeedback(ctx context.Context, userID string, f dto.ScoreFeedbackFilter, limit, offset int) ([]dto.ScoreFeedback, error)
	CountScoreFeedback(ctx context.Context, userID string, f dto.ScoreFeedbackFilter) (current, outdated int, err error)
	DeleteScoreFeedback(ctx context.Context, userID, id string) error
	ClearScoreFeedback(ctx context.Context, userID string) (int64, error)
	GetJobScoreForFeedback(ctx context.Context, userID, jobID string) (dto.JobScoreEvidence, error)
	ListJobScoresForCollection(ctx context.Context, userID string, jobIDs []string) ([]dto.CollectionJobScore, error)
	UpsertGrade(ctx context.Context, userID string, grade dto.Grade) (dto.Grade, error)
	GetGrade(ctx context.Context, userID, jobID string) (dto.Grade, error)
	DeleteGrade(ctx context.Context, userID, jobID string) error
	ListGrades(ctx context.Context, userID string) ([]dto.Grade, error)
	ListImpliedPositives(ctx context.Context, userID string) ([]dto.ImpliedLabel, error)
}

type Service struct {
	store        Store
	answerer     Answerer
	credentials  Credentials
	alerter      Alerter
	pusher       PushSender
	vapidKey     string
	profiles     ProfileReader
	candidates   Reconsiderer
	extractor    Extractor
	tickInterval time.Duration
}

func NewService(deps Deps) *Service {
	tickInterval := deps.TickInterval
	if tickInterval <= 0 {
		tickInterval = defaultAnswerEffectTick
	}
	pusher := deps.Pusher
	if pusher == nil {
		pusher = noopPusher{}
	}
	return &Service{
		store: deps.Store, answerer: deps.Answerer, credentials: deps.Credentials, alerter: deps.Alerter,
		pusher: pusher, vapidKey: deps.VAPIDPublicKey,
		profiles: deps.Profiles, candidates: deps.Candidates, extractor: deps.Extractor, tickInterval: tickInterval,
	}
}

// Run ticks at the configured interval, draining the answer-effect queue.
func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.RunTick(ctx); err != nil && ctx.Err() == nil {
				slog.ErrorContext(ctx, "answer effect tick failed", slog.Any(logger.KeyErr, err))
			}
		}
	}
}

// RunTick drains the answer-effect queue, running up to maxConcurrentEffects at once.
func (s *Service) RunTick(ctx context.Context) error {
	g := &errgroup.Group{}
	g.SetLimit(maxConcurrentEffects)
	for {
		effect, err := s.store.ClaimAnswerEffect(ctx)
		if errors.Is(err, data.ErrNotFound) {
			break
		}
		if err != nil {
			_ = g.Wait()
			return err
		}
		g.Go(func() error {
			if err := s.process(ctx, effect); err != nil {
				slog.ErrorContext(ctx, "answer effect failed", slog.String("effect_id", effect.ID), slog.Any(logger.KeyErr, err))
			}
			return nil
		})
	}
	return g.Wait()
}

func (s *Service) process(ctx context.Context, effect dto.AnswerEffect) error {
	fail := func(err error) error {
		failure := dto.ScoringFailure{Reason: err.Error()}
		var jevErr *jev.Error
		if errors.As(err, &jevErr) {
			failure.Terminal = jevErr.Terminal
			failure.RetryAfter = jevErr.RetryAfter
		}
		if saveErr := s.store.FailAnswerEffect(ctx, effect.ID, effect.Attempts, failure); saveErr != nil {
			return errors.Join(err, saveErr)
		}
		return err
	}

	job, err := s.store.GetJobForScoring(ctx, effect.JobID)
	if err != nil {
		return fail(fmt.Errorf("load answer job: %w", err))
	}
	if job.ContentFingerprint != effect.Fingerprint {
		_, err := s.store.CompleteAnswerEffect(ctx, effect, nil, nil)
		return err
	}

	configs, err := s.store.ListInterestedConfigs(ctx, effect.JobID)
	if err != nil {
		return fail(fmt.Errorf("load interested configs: %w", err))
	}
	surviving := make([]dto.SearchConfig, 0, len(configs))
	for _, cfg := range configs {
		if _, rejected := filter.Reject(job, cfg); !rejected {
			surviving = append(surviving, cfg)
		}
	}
	if len(surviving) == 0 {
		_, err := s.store.CompleteAnswerEffect(ctx, effect, nil, nil)
		return err
	}

	bk, err := s.loadBank(ctx)
	if err != nil {
		return fail(fmt.Errorf("load scoring options: %w", err))
	}
	byID := bk.byID

	asked := make(map[string]string)
	for _, cfg := range surviving {
		for hash, question := range pickedQuestionHashes(cfg.Preferences.Picks, byID) {
			asked[hash] = question
		}
	}

	cached, err := s.store.ListAnswers(ctx, effect.JobID, effect.Fingerprint, jev.Model)
	if err != nil {
		return fail(fmt.Errorf("load cached answers: %w", err))
	}
	corrections, err := s.store.ListJobCorrections(ctx, effect.JobID)
	if err != nil {
		return fail(fmt.Errorf("load corrections: %w", err))
	}

	derived, err := s.profileAnswers(ctx, job, byID, asked)
	if err != nil {
		return fail(err)
	}

	var missing []string
	for hash, question := range asked {
		_, isCached := cached[hash]
		_, isDerived := derived[hash]
		if !isCached && !isDerived {
			missing = append(missing, question)
		}
	}

	fresh, cost, err := s.answerMissing(ctx, surviving, job, missing)
	if err != nil {
		return fail(err)
	}
	maps.Copy(fresh, derived)

	allAnswers := make(map[string]dto.Answer, len(cached)+len(fresh))
	for h, a := range cached {
		allAnswers[h] = a
	}
	for h, a := range fresh {
		allAnswers[h] = a
	}

	scores := make([]dto.JobScore, 0, len(surviving))
	for _, cfg := range surviving {
		score := scoreJob(cfg.UserID, cfg, job, byID, allAnswers, corrections[cfg.UserID], cfg.CompanyIsFavourite)
		score.Cost = cost
		scores = append(scores, score)
	}

	saved, err := s.store.CompleteAnswerEffect(ctx, effect, fresh, scores)
	if err != nil {
		return err
	}
	if !effect.FirstDiscovery {
		return nil
	}

	s.notifyNewJob(ctx, job, surviving, scores, saved)
	return nil
}

func (s *Service) profileAnswers(ctx context.Context, job dto.Job, byID map[string]dto.ScoringOption, asked map[string]string) (map[string]dto.Answer, error) {
	if job.CompanyID == "" {
		return nil, nil
	}
	profile, err := s.store.GetCompanyProfile(ctx, job.CompanyID)
	if errors.Is(err, data.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load company profile: %w", err)
	}
	out := make(map[string]dto.Answer)
	for id, answer := range companyFactAnswers(profile) {
		opt, ok := byID[id]
		if !ok || opt.RetiredAt != nil {
			continue
		}
		if hash := QuestionHash(opt.Question); asked[hash] != "" {
			out[hash] = answer
		}
	}
	return out, nil
}

func (s *Service) answerMissing(ctx context.Context, surviving []dto.SearchConfig, job dto.Job, missing []string) (map[string]dto.Answer, float64, error) {
	fresh := make(map[string]dto.Answer)
	if len(missing) == 0 {
		return fresh, 0, nil
	}
	for _, cfg := range surviving {
		key, err := s.credentials.Get(ctx, cfg.UserID, jev.Provider)
		if err != nil {
			continue
		}
		answers, usage, err := s.answerer.Answer(ctx, key, job, missing)
		if err != nil {
			return nil, 0, fmt.Errorf("answer questions: %w", err)
		}
		for question, a := range answers {
			fresh[QuestionHash(question)] = a
		}
		for _, sc := range surviving {
			slog.InfoContext(ctx, "score call",
				slog.String(logger.KeyEvent, telemetry.EventScoreCall),
				slog.String(logger.KeyUserID, sc.UserID),
				slog.String("model", usage.Model),
				slog.Float64(logger.KeyCostUSD, usage.Cost),
			)
		}
		return fresh, usage.Cost, nil
	}
	return fresh, 0, nil
}

func (s *Service) notifyNewJob(ctx context.Context, job dto.Job, surviving []dto.SearchConfig, scores []dto.JobScore, saved []string) {
	savedSet := make(map[string]bool, len(saved))
	for _, uid := range saved {
		savedSet[uid] = true
	}
	configByUser := make(map[string]dto.SearchConfig, len(surviving))
	for _, cfg := range surviving {
		configByUser[cfg.UserID] = cfg
	}
	for _, sc := range scores {
		cfg := configByUser[sc.UserID]
		if !savedSet[sc.UserID] || cfg.CompanyIsNew || sc.Score < cfg.NotifyThreshold {
			continue
		}
		s.emailNewJob(ctx, job, sc.UserID)
		if err := s.pushToUser(ctx, sc.UserID, newJobPush(job, sc)); err != nil {
			slog.ErrorContext(ctx, "new job push failed", slog.String(logger.KeyUserID, sc.UserID), slog.Any(logger.KeyErr, err))
		}
	}
}

func (s *Service) emailNewJob(ctx context.Context, job dto.Job, userID string) {
	profile, err := s.profiles.GetProfile(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "notification recipient lookup failed", slog.String(logger.KeyUserID, userID), slog.Any(logger.KeyErr, err))
		return
	}
	if profile.Email == "" {
		return
	}
	if err := s.alerter.NotifyNewJob(ctx, job, profile.Email); err != nil {
		slog.ErrorContext(ctx, "notification send failed", slog.String(logger.KeyUserID, userID), slog.Any(logger.KeyErr, err))
	}
}

// Ask answers questions about jobID from the Jev answer cache, sending only
// the misses to Jev with userID's OpenRouter key and caching the results.
// The map is keyed by question text.
func (s *Service) Ask(ctx context.Context, userID, jobID string, questions []string) (map[string]dto.Answer, error) {
	job, err := s.store.GetJobForScoring(ctx, jobID)
	if errors.Is(err, data.ErrNotFound) {
		return nil, apperr.NotFound("job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scoring.Ask: load job: %w", err)
	}

	cached, err := s.store.ListAnswers(ctx, jobID, job.ContentFingerprint, jev.Model)
	if err != nil {
		return nil, fmt.Errorf("scoring.Ask: load cached answers: %w", err)
	}

	out := make(map[string]dto.Answer, len(questions))
	var missing []string
	for _, q := range questions {
		if _, done := out[q]; done {
			continue
		}
		if a, ok := cached[QuestionHash(q)]; ok {
			out[q] = a
			continue
		}
		out[q] = dto.Answer{}
		missing = append(missing, q)
	}
	if len(missing) == 0 {
		return out, nil
	}

	key, err := s.credentials.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return nil, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to answer job questions")
	}
	if err != nil {
		return nil, fmt.Errorf("scoring.Ask: load credential: %w", err)
	}
	answers, usage, err := s.answerer.Answer(ctx, key, job, missing)
	if err != nil {
		return nil, fmt.Errorf("scoring.Ask: answer questions: %w", err)
	}
	slog.InfoContext(ctx, "score call",
		slog.String(logger.KeyEvent, telemetry.EventScoreCall),
		slog.String(logger.KeyUserID, userID),
		slog.String("model", usage.Model),
		slog.Float64(logger.KeyCostUSD, usage.Cost),
	)

	fresh := make(map[string]dto.Answer, len(answers))
	for q, a := range answers {
		out[q] = a
		fresh[QuestionHash(q)] = a
	}
	if err := s.store.SaveAnswers(ctx, jobID, job.ContentFingerprint, jev.Model, fresh); err != nil {
		return nil, fmt.Errorf("scoring.Ask: cache answers: %w", err)
	}
	return out, nil
}

// Recompute re-scores every job userID already has a score for, from their
// current preferences and each job's cached answers. It never calls Answerer.
func (s *Service) Recompute(ctx context.Context, userID string) (dto.RecomputeResult, error) {
	cfg, err := s.searchConfigOrZero(ctx, userID)
	if err != nil {
		return dto.RecomputeResult{}, err
	}

	bk, err := s.loadBank(ctx)
	if err != nil {
		return dto.RecomputeResult{}, err
	}
	byID := bk.byID

	inputs, err := s.store.ListScoringInputs(ctx, userID, jev.Model)
	if err != nil {
		return dto.RecomputeResult{}, err
	}

	scores := scoreInputs(userID, cfg, byID, inputs)
	if err := s.store.SaveScores(ctx, scores); err != nil {
		return dto.RecomputeResult{}, err
	}
	return dto.RecomputeResult{Recomputed: int64(len(scores))}, nil
}

// CompanyFavouriteChanged re-scores userID's scored, open Jobs at companyID
// from cached answers within tx. It never calls Answerer or notifies.
func (s *Service) CompanyFavouriteChanged(ctx context.Context, tx pgx.Tx, userID, companyID string) error {
	cfg, err := s.searchConfigOrZero(ctx, userID)
	if err != nil {
		return err
	}
	bk, err := s.loadBank(ctx)
	if err != nil {
		return err
	}
	inputs, err := s.store.ListCompanyScoringInputs(ctx, tx, userID, companyID, jev.Model)
	if err != nil {
		return err
	}
	return s.store.SaveScoresTx(ctx, tx, scoreInputs(userID, cfg, bk.byID, inputs))
}

// FillMissingAnswers queues an answer effect, without alerting, for each of
// userID's already-scored jobs missing an answer to a currently picked question.
func (s *Service) FillMissingAnswers(ctx context.Context, userID string) (int64, error) {
	cfg, err := s.searchConfigOrZero(ctx, userID)
	if err != nil {
		return 0, err
	}
	if len(cfg.Preferences.Picks) == 0 {
		return 0, nil
	}

	bk, err := s.loadBank(ctx)
	if err != nil {
		return 0, err
	}

	picked := pickedQuestionHashes(cfg.Preferences.Picks, bk.byID)
	if len(picked) == 0 {
		return 0, nil
	}
	hashes := make([]string, 0, len(picked))
	for hash := range picked {
		hashes = append(hashes, hash)
	}

	return s.store.QueueMissingAnswers(ctx, userID, hashes, jev.Model)
}

func (s *Service) loadBank(ctx context.Context) (bank, error) {
	options, err := s.store.ListScoringOptions(ctx)
	if err != nil {
		return bank{}, err
	}
	return newBank(options), nil
}

func (s *Service) searchConfigOrZero(ctx context.Context, userID string) (dto.SearchConfig, error) {
	cfg, err := s.store.GetSearchConfig(ctx, userID)
	if err != nil && !notFound(err) {
		return dto.SearchConfig{}, err
	}
	return cfg, nil
}

func scoreInputs(userID string, cfg dto.SearchConfig, byID map[string]dto.ScoringOption, inputs []store.ScoringInput) []dto.JobScore {
	scores := make([]dto.JobScore, len(inputs))
	for i, in := range inputs {
		scores[i] = scoreJob(userID, cfg, in.Job, byID, in.Answers, in.Corrections, in.Favourite)
	}
	return scores
}

func scoreJob(userID string, cfg dto.SearchConfig, job dto.Job, byID map[string]dto.ScoringOption, answers map[string]dto.Answer, corrections map[string]string, favourite bool) dto.JobScore {
	picks := dedupeBySource(cfg.Preferences.Picks)
	evaluated := evaluatedPicksFor(picks, byID, answers, corrections)
	unpicked := unpickedGateOptions(picks, byID, answers, corrections)
	score, band, rows := compute(evaluated, unpicked, job.SalaryRaw, cfg.Preferences.SalaryFloor, favourite)
	return dto.JobScore{JobID: job.ID, UserID: userID, Score: score, Band: band, Rows: rows, Unknowns: countUnknown(rows)}
}

func evaluatedPicksFor(picks []dto.Pick, byID map[string]dto.ScoringOption, answers map[string]dto.Answer, corrections map[string]string) []evaluatedPick {
	out := make([]evaluatedPick, 0, len(picks))
	for _, p := range picks {
		opt, ok := byID[p.OptionID]
		if !ok {
			continue
		}
		out = append(out, evaluate(opt, p.Stance, answers, corrections))
	}
	return out
}

func unpickedGateOptions(picks []dto.Pick, byID map[string]dto.ScoringOption, answers map[string]dto.Answer, corrections map[string]string) []evaluatedPick {
	pickedIDs := make(map[string]bool, len(picks))
	gateDims := make(map[dto.Dimension]bool)
	for _, p := range picks {
		pickedIDs[p.OptionID] = true
		if opt, ok := byID[p.OptionID]; ok && dimensionSpecs[opt.Dimension].Gate {
			gateDims[opt.Dimension] = true
		}
	}
	var out []evaluatedPick
	for id, opt := range byID {
		if pickedIDs[id] || opt.RetiredAt != nil || !gateDims[opt.Dimension] {
			continue
		}
		out = append(out, evaluate(opt, "nice", answers, corrections))
	}
	return out
}

func evaluate(opt dto.ScoringOption, stance string, answers map[string]dto.Answer, corrections map[string]string) evaluatedPick {
	answer, known := answers[QuestionHash(opt.Question)]
	value, corrected := corrections[opt.ID]
	if corrected {
		answer, known = dto.Answer{PYes: 1}, true
		if value == "no" {
			answer = dto.Answer{PNo: 1}
		}
	}
	return evaluatedPick{
		dimension: opt.Dimension, key: opt.ID, label: opt.Label, stance: stance,
		answer: answer, known: known, retired: opt.RetiredAt != nil, corrected: corrected,
	}
}

func dedupeBySource(picks []dto.Pick) []dto.Pick {
	winners := make(map[string]dto.Pick, len(picks))
	order := make([]string, 0, len(picks))
	for _, p := range picks {
		existing, ok := winners[p.OptionID]
		if !ok {
			winners[p.OptionID] = p
			order = append(order, p.OptionID)
			continue
		}
		if existing.Source != "manual" && p.Source == "manual" {
			winners[p.OptionID] = p
		}
	}
	out := make([]dto.Pick, len(order))
	for i, id := range order {
		out[i] = winners[id]
	}
	return out
}
