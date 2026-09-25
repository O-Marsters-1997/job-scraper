// Package scoringconfig holds the domain rules for reading and updating a
// user's scoring config, including reconsidering existing candidates against
// the new config. Persistence goes through providers.SearchConfigProvider.
package scoringconfig

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

// Reconsiderer re-scores a user's existing candidates against a new config.
type Reconsiderer interface {
	Reconsider(ctx context.Context, config dto.SearchConfig) error
}

type Service struct {
	configs    providers.SearchConfigProvider
	candidates Reconsiderer
}

func New(configs providers.SearchConfigProvider, candidates Reconsiderer) *Service {
	return &Service{configs: configs, candidates: candidates}
}

func (s *Service) Get(ctx context.Context, userID string) (dto.ScoringConfigView, error) {
	cfg, err := s.configs.GetSearchConfig(ctx, userID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		return dto.ScoringConfigView{}, err
	}
	return toView(cfg), nil
}

func (s *Service) Update(ctx context.Context, userID string, in dto.ScoringConfigView) (dto.ScoringConfigView, error) {
	seniority := cleanList(in.ExcludedSeniority)
	for _, level := range seniority {
		if !slices.Contains(score.SeniorityLevels, level) {
			return dto.ScoringConfigView{}, apperr.Invalid("unknown seniority level: " + level)
		}
	}

	cfg := dto.SearchConfig{
		UserID:                userID,
		SuitabilityRubric:     in.SuitabilityRubric,
		NotifyThreshold:       in.NotifyThreshold,
		ExcludedTitleKeywords: cleanList(in.ExcludedTitleKeywords),
		ExcludedCompanies:     cleanList(in.ExcludedCompanies),
		ExcludedSeniority:     seniority,
		ExcludedLocations:     cleanList(in.ExcludedLocations),
	}
	updated, err := s.configs.UpsertSearchConfig(ctx, cfg)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}
	if err := s.candidates.Reconsider(ctx, updated); err != nil {
		return dto.ScoringConfigView{}, fmt.Errorf("reconsider candidates: %w", err)
	}
	return toView(updated), nil
}

func toView(cfg dto.SearchConfig) dto.ScoringConfigView {
	return dto.ScoringConfigView{
		SuitabilityRubric:     cfg.SuitabilityRubric,
		NotifyThreshold:       cfg.NotifyThreshold,
		ExcludedTitleKeywords: nonNilStrings(cfg.ExcludedTitleKeywords),
		ExcludedCompanies:     nonNilStrings(cfg.ExcludedCompanies),
		ExcludedSeniority:     nonNilStrings(cfg.ExcludedSeniority),
		ExcludedLocations:     nonNilStrings(cfg.ExcludedLocations),
	}
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// cleanList trims and lowercases each entry, dropping any that are empty.
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
