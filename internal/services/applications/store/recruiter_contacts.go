package store

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Store) CreateRecruiterContact(ctx context.Context, userID string, in dto.LogRecruiterContactInput) (dto.RecruiterContact, error) {
	panic("not implemented")
}

func (s *Store) ListRecruiterContacts(ctx context.Context, userID, jobID string) ([]dto.RecruiterContact, error) {
	panic("not implemented")
}

func (s *Store) SummarizeRecruiterContacts(ctx context.Context, userID string, jobIDs []string) (map[string]dto.RecruiterContactSummary, error) {
	panic("not implemented")
}
