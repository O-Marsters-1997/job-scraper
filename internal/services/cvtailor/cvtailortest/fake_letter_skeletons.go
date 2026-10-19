package cvtailortest

import (
	"context"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (f *FakeStore) GetLetterSkeleton(_ context.Context, userID string) (dto.LetterSkeleton, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sk, ok := f.skeletons[userID]; ok {
		return sk, nil
	}
	return dto.LetterSkeleton{Sections: []string{}}, nil
}

func (f *FakeStore) SaveLetterSkeleton(_ context.Context, userID string, in dto.LetterSkeleton) (dto.LetterSkeleton, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in.Sections = slices.Clone(in.Sections)
	if in.Sections == nil {
		in.Sections = []string{}
	}
	f.skeletons[userID] = in
	return in, nil
}
