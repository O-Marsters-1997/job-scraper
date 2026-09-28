package aiprefs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/aiprefs"
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
		name  string
		creds aiprefs.CredentialLister
		check func(t *testing.T, got dto.AIPrefsView, err error)
	}{
		{
			name:  "no configured providers",
			creds: stubCredLister{},
			check: wantAIPrefs(dto.AIPrefsView{ConfiguredProviders: []string{}, ScoringEnabled: false}),
		},
		{
			name:  "configured provider enables scoring",
			creds: stubCredLister{providers: []string{"openrouter"}},
			check: wantAIPrefs(dto.AIPrefsView{ConfiguredProviders: []string{"openrouter"}, ScoringEnabled: true}),
		},
		{
			name:  "credential list error propagates",
			creds: stubCredLister{err: errors.New("creds down")},
			check: func(t *testing.T, _ dto.AIPrefsView, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("Get() err = nil, want error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := aiprefs.New(tt.creds)
			got, err := svc.Get(context.Background(), "user-1")
			tt.check(t, got, err)
		})
	}
}

func wantAIPrefs(want dto.AIPrefsView) func(t *testing.T, got dto.AIPrefsView, err error) {
	return func(t *testing.T, got dto.AIPrefsView, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Get() mismatch (-want +got):\n%s", diff)
		}
	}
}
