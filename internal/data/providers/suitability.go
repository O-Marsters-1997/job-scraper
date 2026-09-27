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
	// score, then marks the effect done, all in one transaction.
	CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error)

	ListScoringInputs(ctx context.Context, userID string) ([]ScoringInput, error)
	SaveScores(ctx context.Context, scores []dto.JobScore) error
	GetScoringStatus(ctx context.Context, userID string) (dto.ScoringStatus, error)

	// QueueUserBackfill inserts an answer effect for each of userID's
	// non-closed, already-scored jobs, skipping any with an effect already
	// pending or running. It returns how many were queued.
	QueueUserBackfill(ctx context.Context, userID string) (int64, error)
}
