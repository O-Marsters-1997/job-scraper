package ingest

import (
	"context"
	"log/slog"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Saver interface {
	Save(ctx context.Context, jobs []dto.Job) error
}

// Scorer runs suitability scoring for a specific user after a job is saved and returns the score (0 on error).
// Implementations handle their own errors internally and never block ingest.
type Scorer interface {
	ScoreAndSave(ctx context.Context, job dto.Job, userID string) int
}

// UserLister lists user IDs that have a stored credential for a given provider.
type UserLister interface {
	ListUsersWithProvider(ctx context.Context, provider string) ([]string, error)
}

// CredentialGetter retrieves a plaintext API key for a user+provider pair.
type CredentialGetter interface {
	Get(ctx context.Context, userID, provider string) (string, error)
}

// Notifier sends a per-user notification for a newly ingested job.
// recipientEmail is the user's address; implementations handle errors internally.
type Notifier interface {
	NotifyNewJob(ctx context.Context, job dto.Job, score int, recipientEmail string)
}

// Config wires all Ingester dependencies. Users, Creds, and ScorerFor are all
// required together; omitting any one disables per-user suitability scoring.
type Config struct {
	DB        Saver
	Provider  string // AI provider to fan out to, e.g. "anthropic"
	Users     UserLister
	Creds     CredentialGetter
	ScorerFor func(apiKey string) Scorer
	Emails    providers.UserEmailProvider
	Notifier  Notifier
}

// Ingester is the ingest seam: validate → Save → score (per user) → notify.
type Ingester struct {
	db        Saver
	provider  string
	users     UserLister
	creds     CredentialGetter
	scorerFor func(apiKey string) Scorer
	emails    providers.UserEmailProvider
	notifier  Notifier
}

// New creates an Ingester. Scoring is skipped when Config.Users, Config.Creds,
// or Config.ScorerFor is nil. Notifier and Emails may also be nil.
func New(cfg Config) *Ingester {
	return &Ingester{
		db:        cfg.DB,
		provider:  cfg.Provider,
		users:     cfg.Users,
		creds:     cfg.Creds,
		scorerFor: cfg.ScorerFor,
		emails:    cfg.Emails,
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

	if err := i.db.Save(ctx, valid); err != nil {
		return err
	}
	slog.Info("jobs ingested", slog.Int("count", len(valid)))

	for _, j := range valid {
		i.scoreAndNotifyForAllUsers(ctx, j)
	}
	return nil
}

func (i *Ingester) scoreAndNotifyForAllUsers(ctx context.Context, job dto.Job) {
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
		score := scorer.ScoreAndSave(ctx, job, uid)
		i.notifyUser(ctx, job, score, uid)
	}
}

func (i *Ingester) notifyUser(ctx context.Context, job dto.Job, score int, userID string) {
	if i.notifier == nil || i.emails == nil {
		return
	}
	email, err := i.emails.GetUserEmail(ctx, userID)
	if err != nil {
		slog.Warn("ingest: could not get email for user",
			slog.String("user_id", userID), slog.Any("err", err))
		return
	}
	if email == "" {
		slog.Debug("ingest: skipping notification, user has no email",
			slog.String("user_id", userID))
		return
	}
	i.notifier.NotifyNewJob(ctx, job, score, email)
}

// ProviderForModel resolves the AI provider name for a model ID.
// ponytail: claude-* only; extend when other providers are added.
func ProviderForModel(modelID string) string {
	if strings.HasPrefix(modelID, "claude-") {
		return "anthropic"
	}
	return ""
}
