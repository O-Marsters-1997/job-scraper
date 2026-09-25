// Package aiprefs holds the domain rules for reading and updating a user's AI
// model preferences. Persistence goes through providers.UserAIPrefsProvider;
// which providers a user has configured comes from a CredentialLister.
package aiprefs

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var availableModels = []string{
	"claude-haiku-4-5-20251001",
	"claude-sonnet-4-6",
	"claude-opus-4-8",
}

const (
	defaultSuitabilityModel = "claude-haiku-4-5-20251001"
	defaultReasoningModel   = "claude-sonnet-4-6"
)

// CredentialLister is the subset of credstore.CredentialStore used to report
// which AI providers a user has configured credentials for.
type CredentialLister interface {
	ListProviders(ctx context.Context, userID string) ([]string, error)
}

type Service struct {
	prefs providers.UserAIPrefsProvider
	creds CredentialLister
}

func New(prefs providers.UserAIPrefsProvider, creds CredentialLister) *Service {
	return &Service{prefs: prefs, creds: creds}
}

// Get returns the user's AI prefs, falling back to the defaults when the user
// has none saved yet.
func (s *Service) Get(ctx context.Context, userID string) (dto.AIPrefsView, error) {
	prefs, err := s.prefs.GetUserAIPrefs(ctx, userID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		return dto.AIPrefsView{}, err
	}
	suitabilityModel, reasoningModel := defaultSuitabilityModel, defaultReasoningModel
	if err == nil {
		suitabilityModel, reasoningModel = prefs.SuitabilityModel, prefs.ReasoningModel
	}
	return s.view(ctx, userID, suitabilityModel, reasoningModel)
}

// Update validates and saves the user's model choices, defaulting an empty
// reasoning model, then returns the resulting view.
func (s *Service) Update(ctx context.Context, userID string, in dto.UpdateAIPrefsInput) (dto.AIPrefsView, error) {
	if !isValidModel(in.SuitabilityModel) {
		return dto.AIPrefsView{}, apperr.Invalid("invalid model")
	}
	reasoningModel := in.ReasoningModel
	if reasoningModel == "" {
		reasoningModel = defaultReasoningModel
	} else if !isValidModel(reasoningModel) {
		return dto.AIPrefsView{}, apperr.Invalid("invalid reasoning model")
	}
	prefs, err := s.prefs.UpsertUserAIPrefs(ctx, userID, in.SuitabilityModel, reasoningModel)
	if err != nil {
		return dto.AIPrefsView{}, err
	}
	return s.view(ctx, userID, prefs.SuitabilityModel, prefs.ReasoningModel)
}

func (s *Service) view(ctx context.Context, userID, suitabilityModel, reasoningModel string) (dto.AIPrefsView, error) {
	configured, err := s.creds.ListProviders(ctx, userID)
	if err != nil {
		return dto.AIPrefsView{}, err
	}
	if configured == nil {
		configured = []string{}
	}
	return dto.AIPrefsView{
		SuitabilityModel:    suitabilityModel,
		ReasoningModel:      reasoningModel,
		AvailableModels:     availableModels,
		ConfiguredProviders: configured,
		ScoringEnabled:      len(configured) > 0,
	}, nil
}

func isValidModel(model string) bool {
	for _, m := range availableModels {
		if m == model {
			return true
		}
	}
	return false
}
