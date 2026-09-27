package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ScoringInput is one job and its cached answers, keyed by question hash,
// ready for Recompute.
type ScoringInput struct {
	Job     dto.Job
	Answers map[string]dto.Answer
}

// SuitabilityProvider is the single access point for the answer-effect
// outbox, the shared answer cache and job scores.
type SuitabilityProvider interface {
	ClaimAnswerEffect(ctx context.Context) (dto.AnswerEffect, error)
	FailAnswerEffect(ctx context.Context, id string, attempts int, failure dto.ScoringFailure) error
	GetJobForScoring(ctx context.Context, jobID string) (dto.Job, error)
	ListInterestedConfigs(ctx context.Context, jobID string, discovery bool) ([]dto.SearchConfig, error)
	ListAnswers(ctx context.Context, jobID, fingerprint, model string) (map[string]dto.Answer, error)

	// CompleteAnswerEffect writes new answers and every surviving user's
	// score, then marks the effect done, all in one transaction. It returns
	// the user IDs actually saved (nil if the job's fingerprint moved on
	// since the effect was claimed).
	CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error)

	ListScoringInputs(ctx context.Context, userID string) ([]ScoringInput, error)
	SaveScores(ctx context.Context, scores []dto.JobScore) error
	GetScoringStatus(ctx context.Context, userID string) (dto.ScoringStatus, error)
}
