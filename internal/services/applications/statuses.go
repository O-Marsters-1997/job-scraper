package applications

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	minReplyWindowDays = 1
	maxReplyWindowDays = 60
)

func validateStatusInput(in dto.ApplicationStatusInput) error {
	if in.Name == "" || in.Colour == "" {
		return apperr.Invalid("name and colour are required")
	}
	if d := in.ReplyWindowDays; d != nil && (*d < minReplyWindowDays || *d > maxReplyWindowDays) {
		return apperr.Invalid(fmt.Sprintf("reply window must be between %d and %d working days", minReplyWindowDays, maxReplyWindowDays))
	}
	return nil
}

func (s *Service) CreateStatus(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
	if err := validateStatusInput(in); err != nil {
		return dto.ApplicationStatus{}, err
	}
	return s.store.CreateApplicationStatus(ctx, userID, in.Name, in.Colour, in.ReplyWindowDays, in.CountsAsReply)
}

func (s *Service) UpdateStatus(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
	if err := validateStatusInput(in); err != nil {
		return dto.ApplicationStatus{}, err
	}
	current, err := s.statusOrZero(ctx, userID, in.ID)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	if current.SystemKey != "" && in.Name != current.Name {
		return dto.ApplicationStatus{}, apperr.Invalid(current.Name + " can't be renamed")
	}
	return s.store.UpdateApplicationStatus(ctx, in.ID, userID, in.Name, in.Colour, in.ReplyWindowDays, in.CountsAsReply)
}

func (s *Service) statusOrZero(ctx context.Context, userID, id string) (dto.ApplicationStatus, error) {
	all, err := s.store.ListApplicationStatusesByUser(ctx, userID)
	if err != nil {
		return dto.ApplicationStatus{}, err
	}
	for _, st := range all {
		if st.ID == id {
			return st, nil
		}
	}
	return dto.ApplicationStatus{}, nil
}

func (s *Service) DeleteStatus(ctx context.Context, userID, id string) error {
	current, err := s.statusOrZero(ctx, userID, id)
	if err != nil {
		return err
	}
	if current.SystemKey != "" {
		return apperr.Invalid(current.Name + " can't be deleted")
	}
	count, err := s.store.CountApplicationsUsingStatus(ctx, id, userID)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperr.WithFields(apperr.Conflict("status is in use"), map[string]any{"count": count})
	}
	return s.store.DeleteApplicationStatus(ctx, id, userID)
}
