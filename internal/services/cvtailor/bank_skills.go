package cvtailor

import (
	"context"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) CreateBankSkill(ctx context.Context, userID string, in dto.BankSkillInput) (dto.BankSkill, error) {
	in, err := validateBankSkill(in)
	if err != nil {
		return dto.BankSkill{}, err
	}
	return s.store.CreateBankSkill(ctx, userID, in)
}

func (s *Service) UpdateBankSkill(ctx context.Context, userID string, in dto.BankSkillInput) (dto.BankSkill, error) {
	in, err := validateBankSkill(in)
	if err != nil {
		return dto.BankSkill{}, err
	}
	return s.store.UpdateBankSkill(ctx, userID, in)
}

// ReorderBankSkills moves the Bank Skills into the order of in.IDs.
func (s *Service) ReorderBankSkills(ctx context.Context, userID string, in dto.ReorderInput) (struct{}, error) {
	if hasDuplicates(in.IDs) {
		return struct{}{}, apperr.Invalid("ids must not repeat")
	}
	return struct{}{}, s.store.ReorderBankSkills(ctx, userID, in.IDs)
}

func validateBankSkill(in dto.BankSkillInput) (dto.BankSkillInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Category = strings.TrimSpace(in.Category)
	if in.Name == "" {
		return in, apperr.Invalid("name is required")
	}
	return in, nil
}
