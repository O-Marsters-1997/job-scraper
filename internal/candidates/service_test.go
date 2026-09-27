package candidates

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type memoryStore struct {
	cards       map[string]Candidate
	assessments map[string]bool
	pending     map[string]bool
	nextID      int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{cards: map[string]Candidate{}, assessments: map[string]bool{}, pending: map[string]bool{}}
}

func (m *memoryStore) SaveCards(_ context.Context, target dto.SourceTarget, cards []dto.Job) ([]Candidate, error) {
	out := make([]Candidate, 0, len(cards))
	for _, card := range cards {
		card.Source = target.Source
		candidate, ok := m.cards[card.URL]
		if !ok {
			m.nextID++
			candidate = Candidate{ID: fmt.Sprintf("%06d", m.nextID), URL: card.URL, Card: card}
			m.cards[card.URL] = candidate
		}
		out = append(out, candidate)
	}
	return out, nil
}

func (m *memoryStore) ListForUser(_ context.Context, _ string, afterID string, limit int) ([]Candidate, error) {
	out := make([]Candidate, 0, limit)
	for _, candidate := range m.cards {
		if candidate.ID > afterID {
			out = append(out, candidate)
		}
	}
	slices.SortFunc(out, func(a, b Candidate) int { return strings.Compare(a.ID, b.ID) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryStore) Assess(_ context.Context, candidateID, userID string, version time.Time, passes bool) (bool, error) {
	key := candidateID + userID + version.String()
	m.assessments[key] = passes
	if !passes || m.pending[candidateID] {
		return false, nil
	}
	return true, nil
}

func (m *memoryStore) MarkDetailPending(_ context.Context, candidateID string) error {
	m.pending[candidateID] = true
	return nil
}

type memoryQueue struct {
	jobs []dto.QueuedJob
	err  error
}

func (q *memoryQueue) EnqueueJobs(_ context.Context, jobs []dto.QueuedJob) error {
	if q.err != nil {
		return q.err
	}
	q.jobs = append(q.jobs, jobs...)
	return nil
}

func TestCaptureRetriesAfterQueueFailure(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	q := &memoryQueue{err: errors.New("queue unavailable")}
	service := New(store, q)
	target := dto.SourceTarget{ID: "target-1", UserID: "user-1", Source: "wis"}
	card := dto.Job{URL: "https://example.com/1", Title: "Engineer"}
	config := dto.SearchConfig{UserID: target.UserID, UpdatedAt: time.Now().UTC()}
	if err := service.CapturePage(ctx, target, []dto.Job{card}, config); err == nil {
		t.Fatal("expected queue failure")
	}
	if store.pending[store.cards[card.URL].ID] {
		t.Fatal("candidate pending without confirmed detail")
	}
	q.err = nil
	if err := service.Reconsider(ctx, config); err != nil {
		t.Fatal(err)
	}
	if len(q.jobs) != 1 {
		t.Fatalf("retry queued %d details, want 1", len(q.jobs))
	}
}

func TestCaptureRetainsRejectedCardAndReconsiderationQueuesOnce(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	q := &memoryQueue{}
	service := New(store, q)
	target := dto.SourceTarget{ID: "target-1", UserID: "user-1", Source: "wis"}
	card := dto.Job{URL: "https://example.com/1", Title: "Senior Engineer", Location: "United States"}
	first := dto.SearchConfig{UserID: target.UserID, ExcludedLocations: []string{"united states"}, UpdatedAt: time.Now().UTC()}
	if err := service.CapturePage(ctx, target, []dto.Job{card, card}, first); err != nil {
		t.Fatal(err)
	}
	if len(store.cards) != 1 || len(q.jobs) != 0 {
		t.Fatalf("rejected card: cards=%d queued=%d", len(store.cards), len(q.jobs))
	}

	changed := dto.SearchConfig{UserID: target.UserID, UpdatedAt: first.UpdatedAt.Add(time.Second)}
	if err := service.Reconsider(ctx, changed); err != nil {
		t.Fatal(err)
	}
	if err := service.Reconsider(ctx, changed); err != nil {
		t.Fatal(err)
	}
	if len(q.jobs) != 1 || q.jobs[0].URL != card.URL {
		t.Fatalf("queued after reconsideration = %+v", q.jobs)
	}
}
