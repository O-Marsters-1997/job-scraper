package suitability

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

// ProfileReader looks up a user's notification email.
type ProfileReader interface {
	GetProfile(ctx context.Context, userID string) (dto.Profile, error)
}
