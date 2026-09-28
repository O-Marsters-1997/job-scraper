// Package applicationstatusestest is the applicationstatuses feature's test
// double: a map-backed fake of applicationstatuses.Store, proven against the
// real store by RunStoreContract (ADR 0012).
package applicationstatusestest

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses"
)

var defaultStatusNames = []string{"Saved", "Applied", "Interviewing", "Offer", "Rejected"}

// FakeStore is a map-backed applicationstatuses.Store. InUseCounts lets a
// caller prime CountApplicationsUsingStatus for a status id directly: the
// real answer comes from joining the applications table, which this
// feature's store interface has no way to populate.
type FakeStore struct {
	mu          sync.Mutex
	statuses    map[string]dto.ApplicationStatus
	InUseCounts map[string]int64
}

func NewFakeStore() *FakeStore {
	return &FakeStore{statuses: make(map[string]dto.ApplicationStatus), InUseCounts: make(map[string]int64)}
}

func (f *FakeStore) CreateApplicationStatus(_ context.Context, userID, name, colour string) (dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := dto.ApplicationStatus{ID: fmt.Sprintf("status-%d", len(f.statuses)+1), UserID: userID, Name: name, Colour: colour}
	f.statuses[s.ID] = s
	return s, nil
}

func (f *FakeStore) UpdateApplicationStatus(_ context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.statuses[id]
	if !ok || s.UserID != userID {
		return dto.ApplicationStatus{}, apperr.NotFound("status not found")
	}
	s.Name, s.Colour = name, colour
	f.statuses[id] = s
	return s, nil
}

// DeleteApplicationStatus is a no-op for an unknown id, matching the real
// store's unconditional DELETE.
func (f *FakeStore) DeleteApplicationStatus(_ context.Context, id, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.statuses, id)
	return nil
}

func (f *FakeStore) CountApplicationsUsingStatus(_ context.Context, id, _ string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.InUseCounts[id], nil
}

func (f *FakeStore) ListApplicationStatusesByUser(_ context.Context, userID string) ([]dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationStatus{}
	for _, s := range f.statuses {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *FakeStore) SeedDefaultStatuses(_ context.Context, _ pgx.Tx, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, name := range defaultStatusNames {
		s := dto.ApplicationStatus{ID: fmt.Sprintf("status-%d", len(f.statuses)+1), UserID: userID, Name: name}
		f.statuses[s.ID] = s
	}
	return nil
}

var _ applicationstatuses.Store = (*FakeStore)(nil)
