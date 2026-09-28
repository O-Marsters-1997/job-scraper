// Package queuetest provides a recording double for code that publishes queue.Task messages.
package queuetest

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/queue"
)

// Recorder implements queue publisher interfaces, recording every published task.
type Recorder struct {
	mu    sync.Mutex
	tasks []queue.Task
}

func NewRecorder() *Recorder { return &Recorder{} }

func (r *Recorder) Publish(_ context.Context, task queue.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks = append(r.tasks, task)
	return nil
}

func (r *Recorder) Tasks() []queue.Task {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]queue.Task(nil), r.tasks...)
}
