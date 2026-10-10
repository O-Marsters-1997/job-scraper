// Package eventstest holds the events context's test doubles (ADR 0012).
package eventstest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type row struct {
	userID string
	event  dto.Event
}

type FakeStore struct {
	mu   sync.Mutex
	rows []row
}

func NewFakeStore() *FakeStore { return &FakeStore{} }

func (f *FakeStore) InsertEvent(_ context.Context, _ pgx.Tx, userID, eventType, subjectID string, props []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, row{userID: userID, event: dto.Event{
		ID:        fmt.Sprintf("event-%d", len(f.rows)+1),
		Type:      eventType,
		SubjectID: subjectID,
		Props:     props,
		CreatedAt: time.Now(),
	}})
	return nil
}

func (f *FakeStore) ListEvents(_ context.Context, userID, eventType string) ([]dto.Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.Event{}
	for _, r := range f.rows {
		if r.userID == userID && (eventType == "" || r.event.Type == eventType) {
			out = append(out, r.event)
		}
	}
	return out, nil
}
