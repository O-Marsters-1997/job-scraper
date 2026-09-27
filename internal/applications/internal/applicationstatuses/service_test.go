package applicationstatuses_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/applications/internal/applicationstatuses"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type fakeStore struct {
	mu          sync.Mutex
	statuses    map[string]dto.ApplicationStatus
	countsInUse map[string]int64
}

func (f *fakeStore) CreateApplicationStatus(_ context.Context, userID, name, colour string) (dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.statuses == nil {
		f.statuses = make(map[string]dto.ApplicationStatus)
	}
	s := dto.ApplicationStatus{ID: fmt.Sprintf("status-%d", len(f.statuses)), UserID: userID, Name: name, Colour: colour}
	f.statuses[s.ID] = s
	return s, nil
}

func (f *fakeStore) UpdateApplicationStatus(_ context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error) {
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

func (f *fakeStore) DeleteApplicationStatus(_ context.Context, id, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.statuses, id)
	return nil
}

func (f *fakeStore) CountApplicationsUsingStatus(_ context.Context, id, _ string) (int64, error) {
	return f.countsInUse[id], nil
}

func assertKind(t *testing.T, err error, want apperr.Kind) {
	t.Helper()
	status, ok := apperr.StatusFor(err)
	if !ok || status != want.Status() {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, want.Status())
	}
}

func TestCreateRequiresNameAndColour(t *testing.T) {
	svc := applicationstatuses.New(&fakeStore{})
	_, err := svc.Create(context.Background(), "user-1", dto.ApplicationStatusInput{Name: "Offer"})
	assertKind(t, err, apperr.KindInvalid)
}

func TestCreateSucceeds(t *testing.T) {
	svc := applicationstatuses.New(&fakeStore{})
	got, err := svc.Create(context.Background(), "user-1", dto.ApplicationStatusInput{Name: "Offer", Colour: "#00ff00"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Offer" {
		t.Fatalf("name = %q, want Offer", got.Name)
	}
}

func TestDeleteRefusesAStatusInUse(t *testing.T) {
	store := &fakeStore{countsInUse: map[string]int64{"s1": 3}}
	svc := applicationstatuses.New(store)

	err := svc.Delete(context.Background(), "user-1", "s1")

	assertKind(t, err, apperr.KindConflict)
	fields := apperr.FieldsFor(err)
	if fields["count"] != int64(3) {
		t.Fatalf("fields = %+v, want count=3", fields)
	}
}

func TestDeleteSucceedsWhenUnused(t *testing.T) {
	svc := applicationstatuses.New(&fakeStore{})
	if err := svc.Delete(context.Background(), "user-1", "s1"); err != nil {
		t.Fatal(err)
	}
}
