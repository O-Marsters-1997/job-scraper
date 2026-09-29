// Package cvtailor is the cvtailor context: the Experience Bank of
// Positions and Achievements a User tailors CVs from (ADR 0011).
package cvtailor

import (
	"context"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Store interface {
	ListPositions(ctx context.Context, userID string) ([]dto.Position, error)
	CreatePosition(ctx context.Context, userID string, in dto.PositionInput) (dto.Position, error)
	UpdatePosition(ctx context.Context, userID string, in dto.PositionInput) (dto.Position, error)
	DeletePosition(ctx context.Context, userID, id string) error
	ReorderPositions(ctx context.Context, userID string, ids []string) error
	CreateAchievement(ctx context.Context, userID string, in dto.AchievementInput) (dto.Achievement, error)
	UpdateAchievement(ctx context.Context, userID string, in dto.AchievementInput) (dto.Achievement, error)
	DeleteAchievement(ctx context.Context, userID, id string) error
	ReorderAchievements(ctx context.Context, userID, positionID string, ids []string) error
	ImportPositions(ctx context.Context, userID string, in []dto.ImportPosition) ([]dto.Position, error)
}

type Service struct {
	store Store
	docs  DocFetcher
}

func NewService(store Store, docs DocFetcher) *Service {
	return &Service{store: store, docs: docs}
}

func (s *Service) CreatePosition(ctx context.Context, userID string, in dto.PositionInput) (dto.Position, error) {
	in, err := validatePosition(in)
	if err != nil {
		return dto.Position{}, err
	}
	return s.store.CreatePosition(ctx, userID, in)
}

func (s *Service) UpdatePosition(ctx context.Context, userID string, in dto.PositionInput) (dto.Position, error) {
	in, err := validatePosition(in)
	if err != nil {
		return dto.Position{}, err
	}
	return s.store.UpdatePosition(ctx, userID, in)
}

func (s *Service) CreateAchievement(ctx context.Context, userID string, in dto.AchievementInput) (dto.Achievement, error) {
	in, err := validateAchievement(in)
	if err != nil {
		return dto.Achievement{}, err
	}
	return s.store.CreateAchievement(ctx, userID, in)
}

func (s *Service) UpdateAchievement(ctx context.Context, userID string, in dto.AchievementInput) (dto.Achievement, error) {
	in, err := validateAchievement(in)
	if err != nil {
		return dto.Achievement{}, err
	}
	return s.store.UpdateAchievement(ctx, userID, in)
}

// ReorderPositions moves the Positions into the order of in.IDs.
func (s *Service) ReorderPositions(ctx context.Context, userID string, in dto.ReorderInput) (struct{}, error) {
	if hasDuplicates(in.IDs) {
		return struct{}{}, apperr.Invalid("ids must not repeat")
	}
	return struct{}{}, s.store.ReorderPositions(ctx, userID, in.IDs)
}

// ReorderAchievements moves the Position's Achievements into the order of in.IDs.
func (s *Service) ReorderAchievements(ctx context.Context, userID string, in dto.ReorderInput) (struct{}, error) {
	if hasDuplicates(in.IDs) {
		return struct{}{}, apperr.Invalid("ids must not repeat")
	}
	return struct{}{}, s.store.ReorderAchievements(ctx, userID, in.PositionID, in.IDs)
}

func validatePosition(in dto.PositionInput) (dto.PositionInput, error) {
	in.Employer = strings.TrimSpace(in.Employer)
	in.Title = strings.TrimSpace(in.Title)
	if in.Employer == "" || in.Title == "" {
		return in, apperr.Invalid("employer and title are required")
	}
	in.StartDate = blankToNil(in.StartDate)
	in.EndDate = blankToNil(in.EndDate)
	for _, d := range []*string{in.StartDate, in.EndDate} {
		if d == nil {
			continue
		}
		if _, err := time.Parse(time.DateOnly, *d); err != nil {
			return in, apperr.Invalid("dates must be YYYY-MM-DD")
		}
	}
	if in.StartDate != nil && in.EndDate != nil && *in.EndDate < *in.StartDate {
		return in, apperr.Invalid("end date must not be before start date")
	}
	return in, nil
}

func validateAchievement(in dto.AchievementInput) (dto.AchievementInput, error) {
	in.Text = strings.TrimSpace(in.Text)
	if in.Text == "" {
		return in, apperr.Invalid("text is required")
	}
	return in, nil
}

func blankToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return s
}

func hasDuplicates(ids []string) bool {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}
