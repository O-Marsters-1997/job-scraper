package aiprefs

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

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
		name  string
		creds CredentialLister
		want  dto.AIPrefsView
	}{
		{
			name:  "no configured providers",
			creds: stubCredLister{},
			want:  dto.AIPrefsView{ConfiguredProviders: []string{}, ScoringEnabled: false},
		},
		{
			name:  "configured provider enables scoring",
			creds: stubCredLister{providers: []string{"openrouter"}},
			want:  dto.AIPrefsView{ConfiguredProviders: []string{"openrouter"}, ScoringEnabled: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := New(tt.creds)
			got, err := svc.Get(context.Background(), "user-1")
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Get() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestService_Get_CredentialListErrorPropagates(t *testing.T) {
	t.Parallel()

	svc := New(stubCredLister{err: errors.New("creds down")})
	if _, err := svc.Get(context.Background(), "user-1"); err == nil {
		t.Fatal("Get() err = nil, want error")
	}
}
