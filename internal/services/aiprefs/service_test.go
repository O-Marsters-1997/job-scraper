package aiprefs

import (
	"context"
	"errors"
	"reflect"
	"testing"

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
		name    string
		creds   CredentialLister
		want    dto.AIPrefsView
		wantErr bool
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
		{
			name:    "credential list error propagates",
			creds:   stubCredLister{err: errors.New("creds down")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := New(tt.creds)
			got, err := svc.Get(context.Background(), "user-1")
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %+v; want %+v", got, tt.want)
			}
		})
	}
}
