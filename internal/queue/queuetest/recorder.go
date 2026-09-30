// Package queuetest provides a recording double for code that publishes queue.Task messages.
package queuetest

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

// Recorder implements queue publisher interfaces, recording every published task.
type Recorder struct {
	mu    sync.Mutex
	tasks []queue.Task
	jobs  []dto.QueuedJob
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

func (r *Recorder) EnqueueJobs(_ context.Context, jobs []dto.QueuedJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = append(r.jobs, jobs...)
	return nil
}

func (r *Recorder) Jobs() []dto.QueuedJob {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]dto.QueuedJob(nil), r.jobs...)
}

type publishFails struct {
	*Recorder
	err error
}

func (p publishFails) Publish(context.Context, queue.Task) error { return p.err }

func PublishFails(err error) publishFails {
	return publishFails{Recorder: NewRecorder(), err: err}
}
