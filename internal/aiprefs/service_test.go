package aiprefs

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type stubCredLister struct {
	providers []string
	err       error
}

func (s stubCredLister) ListProviders(_ context.Context, _ string) ([]string, error) {
	return s.providers, s.err
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		seed               func(*providers.MockUserAIPrefsProvider)
		creds              CredentialLister
		userID             string
		wantSuitability    string
		wantReasoning      string
		wantConfigured     []string
		wantScoringEnabled bool
		wantErr            bool
	}{
		{
			name:               "no saved prefs falls back to defaults",
			creds:              stubCredLister{},
			userID:             "user-1",
			wantSuitability:    defaultSuitabilityModel,
			wantReasoning:      defaultReasoningModel,
			wantConfigured:     []string{},
			wantScoringEnabled: false,
		},
		{
			name: "saved prefs override defaults",
			seed: func(m *providers.MockUserAIPrefsProvider) {
				_, _ = m.UpsertUserAIPrefs(context.Background(), "user-1", "claude-opus-4-8", "claude-sonnet-4-6")
			},
			creds:              stubCredLister{providers: []string{"anthropic"}},
			userID:             "user-1",
			wantSuitability:    "claude-opus-4-8",
			wantReasoning:      "claude-sonnet-4-6",
			wantConfigured:     []string{"anthropic"},
			wantScoringEnabled: true,
		},
		{
			name: "prefs lookup error propagates",
			seed: func(m *providers.MockUserAIPrefsProvider) {
				m.GetErr = errors.New("db down")
			},
			creds:   stubCredLister{},
			userID:  "user-1",
			wantErr: true,
		},
		{
			name:    "credential list error propagates",
			creds:   stubCredLister{err: errors.New("creds down")},
			userID:  "user-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := providers.NewMockUserAIPrefsProvider()
			if tt.seed != nil {
				tt.seed(store)
			}
			svc := New(store, tt.creds)

			got, err := svc.Get(context.Background(), tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			want := dto.AIPrefsView{
				SuitabilityModel:    tt.wantSuitability,
				ReasoningModel:      tt.wantReasoning,
				AvailableModels:     availableModels,
				ConfiguredProviders: tt.wantConfigured,
				ScoringEnabled:      tt.wantScoringEnabled,
			}
			if !equalView(got, want) {
				t.Errorf("Get() = %+v; want %+v", got, want)
			}
		})
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		in            dto.UpdateAIPrefsInput
		upsertErr     error
		wantKind      apperr.Kind
		wantErr       bool
		wantErrKind   bool
		wantReasoning string
	}{
		{
			name:        "invalid suitability model",
			in:          dto.UpdateAIPrefsInput{SuitabilityModel: "gpt-4o"},
			wantErr:     true,
			wantErrKind: true,
			wantKind:    apperr.KindInvalid,
		},
		{
			name:        "invalid reasoning model",
			in:          dto.UpdateAIPrefsInput{SuitabilityModel: "claude-sonnet-4-6", ReasoningModel: "gpt-4o"},
			wantErr:     true,
			wantErrKind: true,
			wantKind:    apperr.KindInvalid,
		},
		{
			name:          "empty reasoning model defaults",
			in:            dto.UpdateAIPrefsInput{SuitabilityModel: "claude-sonnet-4-6"},
			wantReasoning: defaultReasoningModel,
		},
		{
			name:          "valid explicit reasoning model kept",
			in:            dto.UpdateAIPrefsInput{SuitabilityModel: "claude-sonnet-4-6", ReasoningModel: "claude-opus-4-8"},
			wantReasoning: "claude-opus-4-8",
		},
		{
			name:      "upsert error propagates",
			in:        dto.UpdateAIPrefsInput{SuitabilityModel: "claude-sonnet-4-6"},
			upsertErr: errors.New("db down"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := providers.NewMockUserAIPrefsProvider()
			store.UpsertErr = tt.upsertErr
			svc := New(store, stubCredLister{})

			got, err := svc.Update(context.Background(), "user-1", tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				if tt.wantErrKind {
					status, ok := apperr.StatusFor(err)
					if !ok {
						t.Fatalf("want kinded error, got %v", err)
					}
					if want := tt.wantKind.Status(); status != want {
						t.Errorf("status = %d; want %d", status, want)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if got.SuitabilityModel != tt.in.SuitabilityModel {
				t.Errorf("SuitabilityModel = %q; want %q", got.SuitabilityModel, tt.in.SuitabilityModel)
			}
			if got.ReasoningModel != tt.wantReasoning {
				t.Errorf("ReasoningModel = %q; want %q", got.ReasoningModel, tt.wantReasoning)
			}
		})
	}
}

func equalView(a, b dto.AIPrefsView) bool {
	if a.SuitabilityModel != b.SuitabilityModel || a.ReasoningModel != b.ReasoningModel || a.ScoringEnabled != b.ScoringEnabled {
		return false
	}
	if len(a.AvailableModels) != len(b.AvailableModels) || len(a.ConfiguredProviders) != len(b.ConfiguredProviders) {
		return false
	}
	for i := range a.AvailableModels {
		if a.AvailableModels[i] != b.AvailableModels[i] {
			return false
		}
	}
	for i := range a.ConfiguredProviders {
		if a.ConfiguredProviders[i] != b.ConfiguredProviders[i] {
			return false
		}
	}
	return true
}
