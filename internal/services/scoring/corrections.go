package scoring

import (
	"context"
	"errors"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

// SetCorrection records userID's claim that the job's answer to an option is
// in.Value, then rescores just that job for them.
func (s *Service) SetCorrection(ctx context.Context, userID string, in dto.CorrectionInput) (dto.JobScore, error) {
	if in.Value != "yes" && in.Value != "no" {
		return dto.JobScore{}, apperr.Invalid("value must be yes or no")
	}
	if err := s.checkCorrectable(ctx, userID, in.JobID, in.OptionID); err != nil {
		return dto.JobScore{}, err
	}
	if err := s.store.SetAnswerCorrection(ctx, userID, in.JobID, in.OptionID, in.Value); err != nil {
		return dto.JobScore{}, fmt.Errorf("scoring.SetCorrection: %w", err)
	}
	return s.rescoreJob(ctx, userID, in.JobID)
}

// RevertCorrection drops userID's Correction of the option and rescores the job.
func (s *Service) RevertCorrection(ctx context.Context, userID string, in dto.RevertCorrectionInput) (dto.JobScore, error) {
	if err := s.checkCorrectable(ctx, userID, in.JobID, in.OptionID); err != nil {
		return dto.JobScore{}, err
	}
	if err := s.store.DeleteAnswerCorrection(ctx, userID, in.JobID, in.OptionID); err != nil {
		return dto.JobScore{}, fmt.Errorf("scoring.RevertCorrection: %w", err)
	}
	return s.rescoreJob(ctx, userID, in.JobID)
}

func (s *Service) checkCorrectable(ctx context.Context, userID, jobID, optionID string) error {
	bk, err := s.loadBank(ctx)
	if err != nil {
		return fmt.Errorf("scoring.checkCorrectable: %w", err)
	}
	if _, ok := bk.byID[optionID]; !ok {
		return apperr.NotFound("option not found")
	}
	_, err = s.store.GetJobScoreForFeedback(ctx, userID, jobID)
	switch {
	case errors.Is(err, data.ErrNotFound):
		return apperr.NotFound("job has no score")
	case err != nil:
		return fmt.Errorf("scoring.checkCorrectable: %w", err)
	}
	return nil
}

func (s *Service) rescoreJob(ctx context.Context, userID, jobID string) (dto.JobScore, error) {
	job, err := s.store.GetJobForScoring(ctx, jobID)
	if err != nil {
		return dto.JobScore{}, fmt.Errorf("scoring.rescoreJob: load job: %w", err)
	}
	cfg, err := s.searchConfigOrZero(ctx, userID)
	if err != nil {
		return dto.JobScore{}, err
	}
	bk, err := s.loadBank(ctx)
	if err != nil {
		return dto.JobScore{}, err
	}
	answers, err := s.store.ListAnswers(ctx, jobID, job.ContentFingerprint, jev.Model)
	if err != nil {
		return dto.JobScore{}, fmt.Errorf("scoring.rescoreJob: load answers: %w", err)
	}
	if len(answers) == 0 {
		return dto.JobScore{}, apperr.Conflict("job is being re-scored, try again shortly")
	}
	corrections, err := s.store.ListJobCorrections(ctx, jobID)
	if err != nil {
		return dto.JobScore{}, fmt.Errorf("scoring.rescoreJob: load corrections: %w", err)
	}
	favourite, err := s.store.IsJobCompanyFavourite(ctx, userID, jobID)
	if err != nil {
		return dto.JobScore{}, fmt.Errorf("scoring.rescoreJob: load favourite: %w", err)
	}
	score := scoreJob(userID, cfg, job, bk.byID, answers, corrections[userID], favourite)
	if err := s.store.SaveScores(ctx, []dto.JobScore{score}); err != nil {
		return dto.JobScore{}, fmt.Errorf("scoring.rescoreJob: %w", err)
	}
	return score, nil
}
