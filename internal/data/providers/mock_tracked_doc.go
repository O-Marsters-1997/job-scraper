package providers

import (
	"context"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockTrackedDocProvider is an in-memory implementation of TrackedDocProvider for tests.
type MockTrackedDocProvider struct {
	docs []dto.TrackedDoc
}

func (m *MockTrackedDocProvider) AddTrackedDoc(_ context.Context, input dto.AddTrackedDocInput) error {
	m.docs = append(m.docs, dto.TrackedDoc{
		ID:      input.UserID + "-" + input.DocID,
		UserID:  input.UserID,
		DocID:   input.DocID,
		AddedAt: time.Now(),
	})
	return nil
}

func (m *MockTrackedDocProvider) RemoveTrackedDoc(_ context.Context, userID, docID string) error {
	for i, d := range m.docs {
		if d.UserID == userID && d.DocID == docID {
			m.docs = append(m.docs[:i], m.docs[i+1:]...)
			return nil
		}
	}
	return ErrTrackedDocNotFound
}

func (m *MockTrackedDocProvider) ListTrackedDocs(_ context.Context, userID string) ([]dto.TrackedDoc, error) {
	var out []dto.TrackedDoc
	for _, d := range m.docs {
		if d.UserID == userID {
			out = append(out, d)
		}
	}
	return out, nil
}

// Seed adds a doc directly to the mock store.
func (m *MockTrackedDocProvider) Seed(d dto.TrackedDoc) {
	m.docs = append(m.docs, d)
}
