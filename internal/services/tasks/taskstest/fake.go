package taskstest

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var defaultTypeNames = []string{"Apply", "Follow up", "Reach out", "Prep"}

func fakeID(n int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012d", n)
}

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

type ownedType struct {
	userID string
	dto.TaskType
}

type ownedTask struct {
	userID string
	dto.Task
}

type FakeStore struct {
	mu    sync.Mutex
	types map[string]ownedType
	tasks map[string]ownedTask
	seq   int
}

func NewFakeStore() *FakeStore {
	return &FakeStore{types: make(map[string]ownedType), tasks: make(map[string]ownedTask)}
}

// AddType creates a Task type for userID; the real store gets its types from
// SeedDefaultTaskTypes or SQL fixtures.
func (f *FakeStore) AddType(userID, name string) dto.TaskType {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	t := dto.TaskType{
		ID:        fakeID(f.seq),
		Name:      name,
		Colour:    "#6366f1",
		CreatedAt: epoch.Add(time.Duration(f.seq) * time.Second),
	}
	f.types[t.ID] = ownedType{userID: userID, TaskType: t}
	return t
}

func (f *FakeStore) SeedDefaultTaskTypes(_ context.Context, _ pgx.Tx, userID string) error {
	for _, name := range defaultTypeNames {
		f.AddType(userID, name)
	}
	return nil
}

func (f *FakeStore) ListTaskTypes(_ context.Context, userID string) ([]dto.TaskType, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.TaskType{}
	for _, t := range f.types {
		if t.userID == userID {
			out = append(out, t.TaskType)
		}
	}
	slices.SortFunc(out, func(a, b dto.TaskType) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}

func (f *FakeStore) ListTasks(_ context.Context, userID string, opts dto.TaskPageOptions) (dto.TaskPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	matched := []dto.Task{}
	for _, t := range f.tasks {
		if t.userID != userID || (opts.TypeID != "" && t.TypeID != opts.TypeID) ||
			(opts.JobID != "" && (t.JobID == nil || *t.JobID != opts.JobID)) ||
			(t.CompletedAt != nil) != opts.Done {
			continue
		}
		matched = append(matched, t.Task)
	}
	slices.SortFunc(matched, func(a, b dto.Task) int { return b.CreatedAt.Compare(a.CreatedAt) })
	start := min(int(opts.Offset), len(matched))
	end := min(start+int(opts.Limit), len(matched))
	return dto.TaskPage{Items: matched[start:end], Total: len(matched)}, nil
}

func (f *FakeStore) CreateTask(_ context.Context, userID string, in dto.CreateTaskInput) (dto.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	typ, ok := f.types[in.TypeID]
	if !ok || typ.userID != userID {
		return dto.Task{}, data.ErrNotFound
	}
	f.seq++
	at := epoch.Add(time.Duration(f.seq) * time.Second)
	t := dto.Task{
		ID:         fakeID(f.seq),
		TypeID:     typ.ID,
		TypeName:   typ.Name,
		TypeColour: typ.Colour,
		JobID:      in.JobID,
		Link:       in.Link,
		Notes:      in.Notes,
		CreatedAt:  at,
		UpdatedAt:  at,
	}
	f.tasks[t.ID] = ownedTask{userID: userID, Task: t}
	return t, nil
}

func (f *FakeStore) SetTaskDone(_ context.Context, userID, id string, done bool, now time.Time) (dto.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok || t.userID != userID {
		return dto.Task{}, data.ErrNotFound
	}
	switch {
	case !done:
		t.CompletedAt = nil
	case t.CompletedAt == nil:
		t.CompletedAt = &now
	}
	t.UpdatedAt = now
	f.tasks[id] = t
	return t.Task, nil
}
