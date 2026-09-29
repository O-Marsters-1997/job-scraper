package cvtailortest

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type headingKey struct{ userID, docID, tabID, heading string }

func (f *FakeStore) ListHeadingMappings(_ context.Context, userID, docID, tabID string) ([]dto.HeadingMapping, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.HeadingMapping{}
	for k, pid := range f.mappings {
		if k.userID == userID && k.docID == docID && k.tabID == tabID {
			out = append(out, dto.HeadingMapping{HeadingText: k.heading, PositionID: pid})
		}
	}
	return out, nil
}

func (f *FakeStore) SaveHeadingMappings(_ context.Context, userID, docID, tabID string, mappings []dto.HeadingMapping) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range mappings {
		f.mappings[headingKey{userID, docID, tabID, m.HeadingText}] = m.PositionID
	}
	return nil
}
