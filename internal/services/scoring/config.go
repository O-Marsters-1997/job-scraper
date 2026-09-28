package scoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

func notFound(err error) bool {
	ae, ok := errors.AsType[*apperr.Error](err)
	return ok && ae.Kind() == apperr.KindNotFound
}

func (s *Service) GetConfig(ctx context.Context, userID string) (dto.ScoringConfigView, error) {
	cfg, err := s.store.GetSearchConfig(ctx, userID)
	if err != nil && !notFound(err) {
		return dto.ScoringConfigView{}, err
	}
	return toView(cfg), nil
}

func (s *Service) UpdateConfig(ctx context.Context, userID string, in dto.ScoringConfigView) (dto.ScoringConfigView, error) {
	if in.NotifyThreshold < 0 || in.NotifyThreshold > 100 {
		return dto.ScoringConfigView{}, apperr.Invalid("notify threshold must be between 0 and 100")
	}
	options, err := s.store.ListScoringOptions(ctx)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}
	b := newBank(options)

	manualPicks, err := validatedPicks(b, in.Preferences.Picks)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}
	floor, err := validatedSalaryFloor(in.Preferences.SalaryFloor)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}

	existing, err := s.store.GetSearchConfig(ctx, userID)
	if err != nil && !notFound(err) {
		return dto.ScoringConfigView{}, err
	}

	text := strings.TrimSpace(in.Preferences.PreferenceText)
	textPicks, hash, err := s.textPicks(ctx, userID, text, existing.Preferences, b)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}

	cfg := dto.SearchConfig{
		UserID:                userID,
		NotifyThreshold:       in.NotifyThreshold,
		ExcludedTitleKeywords: cleanList(in.ExcludedTitleKeywords),
		ExcludedCompanies:     cleanList(in.ExcludedCompanies),
		ExcludedLocations:     cleanList(in.ExcludedLocations),
		Preferences: dto.Preferences{
			Picks:              append(manualPicks, textPicks...),
			SalaryFloor:        floor,
			PreferenceText:     text,
			PreferenceTextHash: hash,
		},
	}
	updated, err := s.store.UpsertSearchConfig(ctx, cfg)
	if err != nil {
		return dto.ScoringConfigView{}, err
	}
	if err := s.candidates.Reconsider(ctx, updated); err != nil {
		return dto.ScoringConfigView{}, fmt.Errorf("reconsider candidates: %w", err)
	}
	if _, err := s.Recompute(ctx, userID); err != nil {
		return dto.ScoringConfigView{}, fmt.Errorf("recompute scores: %w", err)
	}
	queued, err := s.FillMissingAnswers(ctx, userID)
	if err != nil {
		return dto.ScoringConfigView{}, fmt.Errorf("fill missing answers: %w", err)
	}
	view := toView(updated)
	view.BackfillQueued = queued
	return view, nil
}

func (s *Service) textPicks(ctx context.Context, userID, text string, existing dto.Preferences, b bank) ([]dto.Pick, string, error) {
	hash := hashText(text)
	if hash == existing.PreferenceTextHash {
		return existingTextPicks(existing.Picks), hash, nil
	}
	if text == "" {
		return nil, hash, nil
	}
	apiKey, err := s.credentials.Get(ctx, userID, jev.Provider)
	if err != nil {
		return nil, "", apperr.Unprocessable("connect an OpenRouter credential to extract preferences from text")
	}
	extracted, err := s.extractor.Extract(ctx, apiKey, text, b.live, Dimensions)
	if err != nil {
		return nil, "", fmt.Errorf("extract preferences: %w", err)
	}
	picks := make([]dto.Pick, 0, len(extracted))
	for _, p := range extracted {
		opt, ok := b.byID[p.OptionID]
		if !ok || opt.RetiredAt != nil || !stanceAllowed(b.dimensions[opt.Dimension], p.Stance) {
			continue
		}
		picks = append(picks, dto.Pick{OptionID: p.OptionID, Stance: p.Stance, Source: "text"})
	}
	return picks, hash, nil
}

func existingTextPicks(picks []dto.Pick) []dto.Pick {
	var out []dto.Pick
	for _, p := range picks {
		if p.Source == "text" {
			out = append(out, p)
		}
	}
	return out
}

func hashText(text string) string {
	if text == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

type bank struct {
	byID       map[string]dto.ScoringOption
	dimensions map[dto.Dimension]dto.DimensionSpec
	live       []dto.ScoringOption
}

func newBank(options []dto.ScoringOption) bank {
	byID := make(map[string]dto.ScoringOption, len(options))
	live := make([]dto.ScoringOption, 0, len(options))
	for _, o := range options {
		byID[o.ID] = o
		if o.RetiredAt == nil {
			live = append(live, o)
		}
	}
	dimensions := make(map[dto.Dimension]dto.DimensionSpec, len(Dimensions))
	for _, d := range Dimensions {
		dimensions[d.Key] = d
	}
	return bank{byID: byID, dimensions: dimensions, live: live}
}

func validatedPicks(b bank, picks []dto.Pick) ([]dto.Pick, error) {
	out := make([]dto.Pick, len(picks))
	for i, p := range picks {
		opt, ok := b.byID[p.OptionID]
		if !ok || opt.RetiredAt != nil {
			return nil, apperr.Invalid("unknown or retired option: " + p.OptionID)
		}
		if !stanceAllowed(b.dimensions[opt.Dimension], p.Stance) {
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
			Picks:          picksView(cfg.Preferences.Picks),
			SalaryFloor:    cfg.Preferences.SalaryFloor,
			PreferenceText: cfg.Preferences.PreferenceText,
		},
		ExcludedTitleKeywords: nonNilStrings(cfg.ExcludedTitleKeywords),
		ExcludedCompanies:     nonNilStrings(cfg.ExcludedCompanies),
		ExcludedLocations:     nonNilStrings(cfg.ExcludedLocations),
		NotifyThreshold:       cfg.NotifyThreshold,
		UpdatedAt:             cfg.UpdatedAt,
	}
}

func picksView(picks []dto.Pick) []dto.Pick {
	if picks == nil {
		return []dto.Pick{}
	}
	manual := make(map[string]bool, len(picks))
	for _, p := range picks {
		if p.Source == "manual" {
			manual[p.OptionID] = true
		}
	}
	out := make([]dto.Pick, len(picks))
	for i, p := range picks {
		out[i] = p
		out[i].Overridden = p.Source == "text" && manual[p.OptionID]
	}
	return out
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
