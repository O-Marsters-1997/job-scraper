package scoring

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// Answerer asks Jev (or a fake, in tests) to answer a job's missing bank and
// custom questions in one call, billed to apiKey.
type Answerer interface {
	Answer(ctx context.Context, apiKey string, job dto.Job, questions []string) (map[string]dto.Answer, dto.Usage, error)
}

// Credentials reads a user's stored provider API key.
type Credentials interface {
	Get(ctx context.Context, userID, provider string) (string, error)
}

// Alerter sends the new-job notification email.
type Alerter interface {
	NotifyNewJob(ctx context.Context, job dto.Job, email string) error
}

// PushSender delivers one Web Push message; it returns
// notify.ErrSubscriptionGone when the subscription no longer exists.
type PushSender interface {
	Send(ctx context.Context, sub dto.PushSubscriptionInput, msg dto.PushMessage) error
}

// ProfileReader looks up a user's notification email.
type ProfileReader interface {
	GetProfile(ctx context.Context, userID string) (dto.Profile, error)
}

// Extractor turns a user's free preference text into stances against the
// live option bank, billed to apiKey. It may return a pick for an id or
// stance the bank doesn't have; the caller drops those.
type Extractor interface {
	Extract(ctx context.Context, apiKey, text string, options []dto.ScoringOption, dimensions []dto.DimensionSpec) ([]dto.Pick, error)
}

// Reconsiderer re-evaluates a user's discovered-job candidates against an
// updated Search Config.
type Reconsiderer interface {
	Reconsider(ctx context.Context, config dto.SearchConfig) error
}
