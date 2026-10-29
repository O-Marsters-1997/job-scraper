package scoring

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const maxCustomOptions = 10

func (s *Service) CreateCustomOption(ctx context.Context, userID string, in dto.CustomOptionInput) (dto.CustomOption, error) {
	opt, err := validatedCustomOption(in)
	if err != nil {
		return dto.CustomOption{}, err
	}
	live, err := s.store.CountLiveCustomOptions(ctx, userID)
	if err != nil {
		return dto.CustomOption{}, fmt.Errorf("scoring.CreateCustomOption: %w", err)
	}
	if live >= maxCustomOptions {
		return dto.CustomOption{}, apperr.Conflict(fmt.Sprintf("at most %d custom options", maxCustomOptions))
	}
	if err := s.store.AddCustomOption(ctx, userID, opt); err != nil {
		return dto.CustomOption{}, fmt.Errorf("scoring.CreateCustomOption: %w", err)
	}
	if err := s.queueCustomBackfill(ctx, userID, opt); err != nil {
		return dto.CustomOption{}, fmt.Errorf("scoring.CreateCustomOption: %w", err)
	}
	return opt, nil
}

func (s *Service) UpdateCustomOption(ctx context.Context, userID string, in dto.CustomOptionInput) (dto.CustomOption, error) {
	opt, err := validatedCustomOption(in)
	if err != nil {
		return dto.CustomOption{}, err
	}
	bk, err := s.loadBank(ctx, userID)
	if err != nil {
		return dto.CustomOption{}, fmt.Errorf("scoring.UpdateCustomOption: %w", err)
	}
	if old, ok := bk.byID[in.ID]; !ok || old.OwnerID != userID {
		return dto.CustomOption{}, apperr.NotFound("custom option not found")
	}
	cfg, err := s.searchConfigOrDefault(ctx, userID)
	if err != nil {
		return dto.CustomOption{}, err
	}
	for _, p := range cfg.Preferences.Picks {
		if p.OptionID == in.ID && !stanceAllowed(dimensionSpecs[opt.Dimension], p.Stance) {
			return dto.CustomOption{}, apperr.Invalid("stance not allowed for dimension " + string(opt.Dimension) + ": " + string(p.Stance))
		}
	}
	err = s.store.ReplaceCustomOption(ctx, userID, in.ID, opt)
	if notFound(err) {
		return dto.CustomOption{}, apperr.NotFound("custom option not found")
	}
	if err != nil {
		return dto.CustomOption{}, fmt.Errorf("scoring.UpdateCustomOption: %w", err)
	}
	if err := s.queueCustomBackfill(ctx, userID, opt); err != nil {
		return dto.CustomOption{}, fmt.Errorf("scoring.UpdateCustomOption: %w", err)
	}
	return opt, nil
}

func (s *Service) DeleteCustomOption(ctx context.Context, userID, id string) error {
	err := s.store.RetireCustomOption(ctx, userID, id)
	if notFound(err) {
		return apperr.NotFound("custom option not found")
	}
	if err != nil {
		return fmt.Errorf("scoring.DeleteCustomOption: %w", err)
	}
	if _, err := s.Recompute(ctx, userID); err != nil {
		return fmt.Errorf("scoring.DeleteCustomOption: recompute: %w", err)
	}
	return nil
}

func (s *Service) queueCustomBackfill(ctx context.Context, userID string, opt dto.CustomOption) error {
	_, err := s.store.QueueMissingAnswers(ctx, userID, []string{QuestionHash(opt.Question)}, jev.Model)
	return err
}

func validatedCustomOption(in dto.CustomOptionInput) (dto.CustomOption, error) {
	if _, ok := dimensionSpecs[in.Dimension]; !ok {
		return dto.CustomOption{}, apperr.Invalid("unknown dimension: " + string(in.Dimension))
	}
	if in.Dimension == dto.DimensionSeniority {
		return dto.CustomOption{}, apperr.Invalid("custom options are not allowed in seniority")
	}
	label, question := strings.TrimSpace(in.Label), strings.TrimSpace(in.Question)
	if label == "" || question == "" {
		return dto.CustomOption{}, apperr.Invalid("label and question are required")
	}
	return dto.CustomOption{ID: "custom:" + uuid.NewString(), Dimension: in.Dimension, Label: label, Question: question}, nil
}
