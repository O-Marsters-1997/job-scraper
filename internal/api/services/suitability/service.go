package suitability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/ollymarsters/job-scraper/internal/api/jev"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/filter"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

const maxConcurrentEffects = 4

type Service struct {
	store       providers.SuitabilityProvider
	options     providers.ScoringOptionsProvider
	configs     providers.SearchConfigProvider
	answerer    Answerer
	credentials Credentials
	alerter     Alerter
	profiles    ProfileReader
}

func New(store providers.SuitabilityProvider, options providers.ScoringOptionsProvider, configs providers.SearchConfigProvider, answerer Answerer, credentials Credentials, alerter Alerter, profiles ProfileReader) *Service {
	return &Service{store: store, options: options, configs: configs, answerer: answerer, credentials: credentials, alerter: alerter, profiles: profiles}
}

// RunTick drains the answer-effect queue, running up to maxConcurrentEffects at once.
func (s *Service) RunTick(ctx context.Context) error {
	g := &errgroup.Group{}
	g.SetLimit(maxConcurrentEffects)
	for {
		effect, err := s.store.ClaimAnswerEffect(ctx)
		if errors.Is(err, providers.ErrNotFound) {
			break
		}
		if err != nil {
			_ = g.Wait()
			return err
		}
		g.Go(func() error {
			if err := s.process(ctx, effect); err != nil {
				slog.Error("answer effect failed", slog.String("effect_id", effect.ID), slog.Any("err", err))
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
			failure.Terminal = jevErr.Kind == jev.FailureTerminal
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

	discovery, _ := sourcespec.SourceRole(job.Source)
	configs, err := s.store.ListInterestedConfigs(ctx, effect.JobID, discovery == sourcespec.RoleDiscovery)
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

	options, err := s.options.ListScoringOptions(ctx)
	if err != nil {
		return fail(fmt.Errorf("load scoring options: %w", err))
	}
	byID := optionsByID(options)
	questionByHash := make(map[string]string, len(options))
	for _, o := range options {
		questionByHash[questionHash(o.Question)] = o.Question
	}

	cached, err := s.store.ListAnswers(ctx, effect.JobID, effect.Fingerprint, jev.Model)
	if err != nil {
		return fail(fmt.Errorf("load cached answers: %w", err))
	}

	var missing []string
	for hash, question := range questionByHash {
		if _, ok := cached[hash]; !ok {
			missing = append(missing, question)
		}
	}

	fresh := make(map[string]dto.Answer)
	var cost float64
	if len(missing) > 0 {
		for _, cfg := range surviving {
			key, err := s.credentials.Get(ctx, cfg.UserID, jev.Provider)
			if err != nil {
				continue
			}
			answers, usage, err := s.answerer.Answer(ctx, key, job, missing)
			if err != nil {
				return fail(fmt.Errorf("answer questions: %w", err))
			}
			for question, a := range answers {
				fresh[questionHash(question)] = a
			}
			cost = usage.Cost
			break
		}
	}

	allAnswers := make(map[string]dto.Answer, len(cached)+len(fresh))
	for h, a := range cached {
		allAnswers[h] = a
	}
	for h, a := range fresh {
		allAnswers[h] = a
	}

	scores := make([]dto.JobScore, 0, len(surviving))
	for _, cfg := range surviving {
		picks := evaluatedPicksFor(cfg.Preferences.Picks, byID, allAnswers)
		score, rows := compute(picks)
		scores = append(scores, dto.JobScore{JobID: effect.JobID, UserID: cfg.UserID, Score: score, Rows: rows, Unknowns: countUnknown(rows), Cost: cost})
	}

	saved, err := s.store.CompleteAnswerEffect(ctx, effect, fresh, scores)
	if err != nil {
		return err
	}
	if !effect.FirstDiscovery {
		return nil
	}

	savedSet := make(map[string]bool, len(saved))
	for _, uid := range saved {
		savedSet[uid] = true
	}
	configByUser := make(map[string]dto.SearchConfig, len(surviving))
	for _, cfg := range surviving {
		configByUser[cfg.UserID] = cfg
	}
	for _, sc := range scores {
		if !savedSet[sc.UserID] || sc.Score < configByUser[sc.UserID].NotifyThreshold {
			continue
		}
		profile, err := s.profiles.GetProfile(ctx, sc.UserID)
		if err != nil {
			slog.Error("notification recipient lookup failed", slog.String("user_id", sc.UserID), slog.Any("err", err))
			continue
		}
		if profile.Email == "" {
			continue
		}
		if err := s.alerter.NotifyNewJob(ctx, job, profile.Email); err != nil {
			slog.Error("notification send failed", slog.String("user_id", sc.UserID), slog.Any("err", err))
		}
	}
	return nil
}

// Recompute re-scores every job userID already has a score for, from their
// current preferences and each job's cached answers. It never calls Answerer.
func (s *Service) Recompute(ctx context.Context, userID string) (dto.RecomputeResult, error) {
	cfg, err := s.configs.GetSearchConfig(ctx, userID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		return dto.RecomputeResult{}, err
	}

	options, err := s.options.ListScoringOptions(ctx)
	if err != nil {
		return dto.RecomputeResult{}, err
	}
	byID := optionsByID(options)

	inputs, err := s.store.ListScoringInputs(ctx, userID)
	if err != nil {
		return dto.RecomputeResult{}, err
	}

	scores := make([]dto.JobScore, len(inputs))
	for i, in := range inputs {
		picks := evaluatedPicksFor(cfg.Preferences.Picks, byID, in.Answers)
		score, rows := compute(picks)
		scores[i] = dto.JobScore{JobID: in.Job.ID, UserID: userID, Score: score, Rows: rows, Unknowns: countUnknown(rows)}
	}
	if err := s.store.SaveScores(ctx, scores); err != nil {
		return dto.RecomputeResult{}, err
	}
	return dto.RecomputeResult{Recomputed: int64(len(scores))}, nil
}

func optionsByID(options []dto.ScoringOption) map[string]dto.ScoringOption {
	byID := make(map[string]dto.ScoringOption, len(options))
	for _, o := range options {
		byID[o.ID] = o
	}
	return byID
}

// evaluatedPicksFor joins picks with the bank (for dimension and label) and
// the job's cached answers (for the resolved value). A pick on an unknown
// option id is skipped: it cannot be scored or shown.
func evaluatedPicksFor(picks []dto.Pick, byID map[string]dto.ScoringOption, answers map[string]dto.Answer) []evaluatedPick {
	out := make([]evaluatedPick, 0, len(picks))
	for _, p := range picks {
		opt, ok := byID[p.OptionID]
		if !ok {
			continue
		}
		answer, known := answers[questionHash(opt.Question)]
		out = append(out, evaluatedPick{
			dimension: opt.Dimension, key: opt.ID, label: opt.Label, stance: p.Stance,
			answer: answer, known: known,
		})
	}
	return out
}
