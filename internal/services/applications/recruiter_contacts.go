package applications

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) LogRecruiterContact(ctx context.Context, userID string, in dto.LogRecruiterContactInput) (dto.RecruiterContact, error) {
	panic("not implemented")
}

func (s *Service) ListRecruiterContacts(ctx context.Context, userID string, q dto.RecruiterContactsQuery) ([]dto.RecruiterContact, error) {
	panic("not implemented")
}

func (s *Service) RecruiterContactsForJobs(ctx context.Context, userID string, q dto.RecruiterContactsForJobsQuery) (map[string]dto.RecruiterContactSummary, error) {
	panic("not implemented")
}
