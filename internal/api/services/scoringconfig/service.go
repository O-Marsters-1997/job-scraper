package scoringconfig

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Reconsiderer interface {
	Reconsider(ctx context.Context, config dto.SearchConfig) error
}

type Recomputer interface {
	Recompute(ctx context.Context, userID string) (dto.RecomputeResult, error)
}

type Service struct {
	configs    providers.SearchConfigProvider
	candidates Reconsiderer
	options    providers.ScoringOptionsProvider
	recompute  Recomputer
}

func New(configs providers.SearchConfigProvider, candidates Reconsiderer, options providers.ScoringOptionsProvider, recompute Recomputer) *Service {
	return &Service{configs: configs, candidates: candidates, options: options, recompute: recompute}
}

func (s *Service) Get(ctx context.Context, userID string) (dto.ScoringConfigView, error) {
	cfg, err := s.configs.GetSearchConfig(ctx, userID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		return dto.ScoringConfigView{}, err
	}
	return toView(cfg), nil
}

func (s *Service) Update(ctx context.Context, userID string, in dto.ScoringConfigView) (dto.ScoringConfigView, error) {
	if in.NotifyThreshold < 0 || in.NotifyThreshold > 100 {
		return dto.ScoringConfigView{}, apperr.Invalid("notify threshold must be between 0 and 100")
	}
	picks, err := s.validatedPicks(ctx, in.Preferences.Picks)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}
	floor, err := validatedSalaryFloor(in.Preferences.SalaryFloor)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}

	cfg := dto.SearchConfig{
		UserID:            userID,
		NotifyThreshold:   in.NotifyThreshold,
		ExcludedCompanies: cleanList(in.ExcludedCompanies),
		ExcludedLocations: cleanList(in.ExcludedLocations),
		Preferences: dto.Preferences{
			Picks:       picks,
			SalaryFloor: floor,
			BlockedTech: cleanList(in.Preferences.BlockedTech),
		},
	}
	updated, err := s.configs.UpsertSearchConfig(ctx, cfg)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}
	if err := s.candidates.Reconsider(ctx, updated); err != nil {
		return dto.ScoringConfigView{}, fmt.Errorf("reconsider candidates: %w", err)
	}
	if _, err := s.recompute.Recompute(ctx, userID); err != nil {
		return dto.ScoringConfigView{}, fmt.Errorf("recompute scores: %w", err)
	}
	return toView(updated), nil
}

// validatedPicks rejects an unknown or retired option, or a stance the
// option's dimension doesn't allow, and normalises every surviving pick's
// source to manual: only extraction (not yet built) can write "text".
func (s *Service) validatedPicks(ctx context.Context, picks []dto.Pick) ([]dto.Pick, error) {
	options, err := s.options.ListScoringOptions(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]dto.ScoringOption, len(options))
	for _, o := range options {
		byID[o.ID] = o
	}
	dimensions := make(map[dto.Dimension]dto.DimensionSpec, len(Dimensions))
	for _, d := range Dimensions {
		dimensions[d.Key] = d
	}

	out := make([]dto.Pick, len(picks))
	for i, p := range picks {
		opt, ok := byID[p.OptionID]
		if !ok || opt.RetiredAt != nil {
			return nil, apperr.Invalid("unknown or retired option: " + p.OptionID)
		}
		spec := dimensions[opt.Dimension]
		if !stanceAllowed(spec, p.Stance) {
			return nil, apperr.Invalid("stance not allowed for dimension " + string(opt.Dimension) + ": " + p.Stance)
		}
		out[i] = dto.Pick{OptionID: p.OptionID, Stance: p.Stance, Source: "manual"}
	}
	return out, nil
}

func validatedSalaryFloor(floor *dto.Money) (*dto.Money, error) {
	if floor == nil {
		return nil, nil
	}
	currency := strings.ToUpper(strings.TrimSpace(floor.Currency))
	if floor.Amount < 0 {
		return nil, apperr.Invalid("salary floor amount must not be negative")
	}
	if currency == "" {
		return nil, apperr.Invalid("salary floor currency is required")
	}
	return &dto.Money{Amount: floor.Amount, Currency: currency}, nil
}

func stanceAllowed(spec dto.DimensionSpec, stance string) bool {
	for _, s := range spec.Stances {
		if s == stance {
			return true
		}
	}
	return false
}

func toView(cfg dto.SearchConfig) dto.ScoringConfigView {
	return dto.ScoringConfigView{
		Preferences: dto.Preferences{
			Picks:       nonNilPicks(cfg.Preferences.Picks),
			SalaryFloor: cfg.Preferences.SalaryFloor,
			BlockedTech: nonNilStrings(cfg.Preferences.BlockedTech),
		},
		ExcludedCompanies: nonNilStrings(cfg.ExcludedCompanies),
		ExcludedLocations: nonNilStrings(cfg.ExcludedLocations),
		NotifyThreshold:   cfg.NotifyThreshold,
		UpdatedAt:         cfg.UpdatedAt,
	}
}

func nonNilPicks(p []dto.Pick) []dto.Pick {
	if p == nil {
		return []dto.Pick{}
	}
	return p
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func cleanList(items []string) []string {
	cleaned := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return cleaned
}
