package score

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type EffectStore interface {
	ClaimScoringEffect(context.Context) (dto.ScoringEffect, error)
	GetJob(context.Context, string, string) (dto.Job, error)
	GetSearchConfig(context.Context, string) (dto.SearchConfig, error)
	GetProfile(context.Context, string) (dto.Profile, error)
	FailScoringEffect(context.Context, string, int, string) error
	CompleteScoringEffect(context.Context, dto.ScoringEffect, int, string, []string, []string) (bool, error)
}

type OutboxWorker struct {
	store     EffectStore
	getKey    func(context.Context, string) (string, error)
	scorerFor func(string) SuitabilityScorer
	sendAlert func(context.Context, dto.Job, string) error
}

func NewOutboxWorker(store EffectStore, getKey func(context.Context, string) (string, error), scorerFor func(string) SuitabilityScorer, sendAlert func(context.Context, dto.Job, string) error) *OutboxWorker {
	return &OutboxWorker{store: store, getKey: getKey, scorerFor: scorerFor, sendAlert: sendAlert}
}

func (w *OutboxWorker) RunOnce(ctx context.Context) error {
	effect, err := w.store.ClaimScoringEffect(ctx)
	if err != nil {
		return err
	}
	fail := func(err error) error {
		if saveErr := w.store.FailScoringEffect(ctx, effect.ID, effect.Attempts, err.Error()); saveErr != nil {
			return errors.Join(err, saveErr)
		}
		return err
	}
	job, err := w.store.GetJob(ctx, effect.JobID, effect.UserID)
	if err != nil {
		return fail(fmt.Errorf("load scoring job: %w", err))
	}
	if job.ContentFingerprint != effect.Fingerprint {
		_, err := w.store.CompleteScoringEffect(ctx, effect, 0, "", nil, nil)
		return err
	}
	cfg, err := w.store.GetSearchConfig(ctx, effect.UserID)
	if errors.Is(err, providers.ErrNotFound) {
		cfg = dto.SearchConfig{UserID: effect.UserID}
	} else if err != nil {
		return fail(fmt.Errorf("load scoring config: %w", err))
	}
	if !cfg.UpdatedAt.Equal(effect.ConfigVersion) && !cfg.UpdatedAt.IsZero() {
		_, err := w.store.CompleteScoringEffect(ctx, effect, 0, "", nil, nil)
		return err
	}
	if !strings.HasPrefix(effect.Model, "claude-") {
		return fail(fmt.Errorf("unsupported scoring model %q", effect.Model))
	}
	key, err := w.getKey(ctx, effect.UserID)
	if err != nil {
		return fail(fmt.Errorf("load scoring credential: %w", err))
	}
	result, err := w.scorerFor(key).Score(ctx, job, cfg, effect.Model)
	if err != nil {
		return fail(fmt.Errorf("score job: %w", err))
	}
	email := ""
	if effect.FirstDiscovery && result.Score >= cfg.NotifyThreshold && w.sendAlert != nil {
		profile, err := w.store.GetProfile(ctx, effect.UserID)
		if err != nil {
			return fail(fmt.Errorf("load notification recipient: %w", err))
		}
		email = profile.Email
	}
	saved, err := w.store.CompleteScoringEffect(ctx, effect, result.Score, result.Rationale, result.Matched, result.Missing)
	if err != nil {
		return err
	}
	if !saved {
		return nil
	}
	if email == "" {
		if effect.FirstDiscovery && result.Score >= cfg.NotifyThreshold && w.sendAlert != nil {
			slog.Debug("notification skipped: user has no email", slog.String("user_id", effect.UserID))
		}
		return nil
	}
	if err := w.sendAlert(ctx, job, email); err != nil {
		slog.Error("notification send failed", slog.String("user_id", effect.UserID), slog.String("job_id", job.ID), slog.Any("err", err))
	}
	return nil
}
