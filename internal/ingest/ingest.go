package ingest

import (
	"context"
	"log/slog"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Saver interface {
	Save(ctx context.Context, jobs []dto.Job) ([]dto.Job, error)
}

// Scorer runs suitability scoring for a batch of jobs for a specific user after they are saved.
// Implementations handle their own errors internally and never block ingest.
type Scorer interface {
	ScoreAndSaveBatch(ctx context.Context, jobs []dto.Job, userID string)
}

type UserLister interface {
	ListUsersWithProvider(ctx context.Context, provider string) ([]string, error)
}

type CredentialGetter interface {
	Get(ctx context.Context, userID, provider string) (string, error)
}

// Notifier sends a notification for a newly ingested job.
// score is the suitability score (0 when no scorer is configured).
// Implementations handle their own errors internally.
type Notifier interface {
	NotifyNewJob(ctx context.Context, job dto.Job, score int)
}

// Config wires all Ingester dependencies. Users, Creds, and ScorerFor are all
// required together; omitting any one disables per-user suitability scoring.
type Config struct {
	DB        Saver
	Provider  string // AI provider to fan out to, e.g. "anthropic"
	Users     UserLister
	Creds     CredentialGetter
	ScorerFor func(apiKey string) Scorer
	Notifier  Notifier
}

// Ingester is the ingest seam: validate → Save → score (per user) → notify.
type Ingester struct {
	db        Saver
	provider  string
	users     UserLister
	creds     CredentialGetter
	scorerFor func(apiKey string) Scorer
	notifier  Notifier
}

// New creates an Ingester. Scoring is skipped when Config.Users, Config.Creds,
// or Config.ScorerFor is nil. Notifier may also be nil.
func New(cfg Config) *Ingester {
	return &Ingester{
		db:        cfg.DB,
		provider:  cfg.Provider,
		users:     cfg.Users,
		creds:     cfg.Creds,
		scorerFor: cfg.ScorerFor,
		notifier:  cfg.Notifier,
	}
}

// Ingest saves valid jobs then fans out suitability scoring to every user that
// has a credential for the configured provider. Save errors are returned;
// scoring and notification failures are fire-and-forget.
func (i *Ingester) Ingest(ctx context.Context, jobs []dto.Job) error {
	valid := make([]dto.Job, 0, len(jobs))
	for _, j := range jobs {
		if j.Title == "" || j.URL == "" {
			slog.Warn("skipping invalid job", slog.String("title", j.Title), slog.String("url", j.URL))
			continue
		}
		valid = append(valid, j)
	}
	if len(valid) == 0 {
		return nil
	}

	saved, err := i.db.Save(ctx, valid)
	if err != nil {
		return err
	}
	slog.Info("jobs ingested", slog.Int("count", len(saved)))

	i.scoreForAllUsers(ctx, saved)

	for _, j := range saved {
		if i.notifier != nil {
			i.notifier.NotifyNewJob(ctx, j, 0)
		}
	}
	return nil
}

func (i *Ingester) scoreForAllUsers(ctx context.Context, jobs []dto.Job) {
	if i.users == nil || i.creds == nil || i.scorerFor == nil || i.provider == "" {
		return
	}

	userIDs, err := i.users.ListUsersWithProvider(ctx, i.provider)
	if err != nil {
		slog.Error("ingest: could not list users with provider",
			slog.String("provider", i.provider), slog.Any("err", err))
		return
	}

	for _, uid := range userIDs {
		apiKey, err := i.creds.Get(ctx, uid, i.provider)
		if err != nil {
			slog.Warn("ingest: could not get credential for user",
				slog.String("user_id", uid), slog.String("provider", i.provider), slog.Any("err", err))
			continue
		}
		scorer := i.scorerFor(apiKey)
		scorer.ScoreAndSaveBatch(ctx, jobs, uid)
	}
}

// ProviderForModel resolves the AI provider name for a model ID.
// ponytail: claude-* only; extend when other providers are added.
func ProviderForModel(modelID string) string {
	if strings.HasPrefix(modelID, "claude-") {
		return "anthropic"
	}
	return ""
}
