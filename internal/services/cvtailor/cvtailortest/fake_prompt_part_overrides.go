package cvtailortest

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/promptparts"
)

type overrideKey struct {
	userID, generation, path string
}

func (f *FakeStore) ListPartOverrides(_ context.Context, userID string, g promptparts.GenerationName) ([]promptparts.Override, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []promptparts.Override
	for k, o := range f.overrides {
		if k.userID == userID && k.generation == string(g) {
			out = append(out, o)
		}
	}
	return out, nil
}

func (f *FakeStore) SavePartOverride(_ context.Context, userID string, g promptparts.GenerationName, o promptparts.Override) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.overrides[overrideKey{userID, string(g), o.Path}] = o
	return nil
}

func (f *FakeStore) DeletePartOverride(_ context.Context, userID string, g promptparts.GenerationName, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.overrides, overrideKey{userID, string(g), path})
	return nil
}
