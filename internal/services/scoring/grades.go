package scoring

import (
	"context"
	"fmt"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	gradeValues  = []string{"great", "ok", "no"}
	gradeReasons = []string{"seniority", "role", "tech", "domain", "location", "salary", "contract", "company_size", "recruiter", "other"}
)

// SetGrade records userID's Grade for in.JobID, replacing any earlier one,
// with the job's stored score and model at that moment.
func (s *Service) SetGrade(ctx context.Context, userID string, in dto.GradeInput) (dto.Grade, error) {
	if !slices.Contains(gradeValues, in.Grade) {
		return dto.Grade{}, apperr.Invalid("grade must be great, ok or no")
	}
	for _, r := range in.Reasons {
		if !slices.Contains(gradeReasons, r) {
			return dto.Grade{}, apperr.Invalid("unknown reason " + r)
		}
	}
	if _, err := s.store.GetJobForScoring(ctx, in.JobID); notFound(err) {
		return dto.Grade{}, apperr.NotFound("job not found")
	} else if err != nil {
		return dto.Grade{}, fmt.Errorf("scoring.SetGrade: load job: %w", err)
	}
	grade := dto.Grade{JobID: in.JobID, Grade: in.Grade, Reasons: slices.Compact(slices.Sorted(slices.Values(in.Reasons)))}
	switch ev, err := s.store.GetJobScoreForFeedback(ctx, userID, in.JobID); {
	case err == nil:
		grade.ScoreAtGrade, grade.ScoreModel = &ev.Score, ev.Model
	case !notFound(err):
		return dto.Grade{}, fmt.Errorf("scoring.SetGrade: load score: %w", err)
	}
	return s.store.UpsertGrade(ctx, userID, grade)
}

// GetGrade returns userID's Grade for jobID, or nil when ungraded.
func (s *Service) GetGrade(ctx context.Context, userID, jobID string) (*dto.Grade, error) {
	g, err := s.store.GetGrade(ctx, userID, jobID)
	if notFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scoring.GetGrade: %w", err)
	}
	return &g, nil
}

// ClearGrade removes userID's Grade for jobID; clearing an ungraded job is
// not an error.
func (s *Service) ClearGrade(ctx context.Context, userID, jobID string) error {
	return s.store.DeleteGrade(ctx, userID, jobID)
}

// ListGrades returns every Grade userID has given, newest first.
func (s *Service) ListGrades(ctx context.Context, userID string) ([]dto.Grade, error) {
	return s.store.ListGrades(ctx, userID)
}
