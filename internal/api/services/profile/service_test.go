package profile_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/api/services/profile"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type fakeStore struct {
	profile dto.Profile
}

func (f *fakeStore) GetProfile(context.Context, string) (dto.Profile, error) {
	return f.profile, nil
}

func (f *fakeStore) UpdateEmail(_ context.Context, _, email string) (dto.Profile, error) {
	f.profile.Email = email
	return f.profile, nil
}

func TestGet(t *testing.T) {
	store := &fakeStore{profile: dto.Profile{Username: "alice", Email: "alice@example.com"}}
	svc := profile.New(store)
	got, err := svc.Get(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	want := dto.ProfileView{Username: "alice", Email: "alice@example.com"}
	if got != want {
		t.Fatalf("got = %+v, want %+v", got, want)
	}
}

func TestUpdate(t *testing.T) {
	store := &fakeStore{profile: dto.Profile{Username: "alice", Email: "old@example.com"}}
	svc := profile.New(store)
	if _, err := svc.Update(context.Background(), "user-1", dto.UpdateProfileInput{Email: "new@example.com"}); err != nil {
		t.Fatal(err)
	}
	if store.profile.Email != "new@example.com" {
		t.Fatalf("email = %q, want new@example.com", store.profile.Email)
	}
}
