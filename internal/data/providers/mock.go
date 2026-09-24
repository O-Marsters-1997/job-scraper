package providers

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// MockJobProvider lives outside _test.go so it can be imported by tests in other packages.
type MockJobProvider struct {
	mu   sync.Mutex
	jobs map[string]dto.Job // keyed by URL

	// Injectable errors for failure-path tests.
	SaveErr    error
	NewURLsErr error
}

func NewMockJobProvider() *MockJobProvider {
	return &MockJobProvider{jobs: make(map[string]dto.Job)}
}

func (m *MockJobProvider) Save(_ context.Context, jobs []dto.Job) ([]dto.Job, error) {
	if m.SaveErr != nil {
		return nil, m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.Job, len(jobs))
	for idx, j := range jobs {
		if j.ID == "" {
			if previous, ok := m.jobs[j.URL]; ok {
				j.ID = previous.ID
			} else {
				j.ID = "mock-" + strconv.Itoa(len(m.jobs)+1)
			}
		}
		m.jobs[j.URL] = j
		out[idx] = j
	}
	return out, nil
}

func (m *MockJobProvider) SaveCanonical(_ context.Context, job dto.Job) (dto.Job, string, error) {
	if m.SaveErr != nil {
		return dto.Job{}, "", m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if previous, ok := m.jobs[job.URL]; ok {
		job.ID = previous.ID
		if job.Title == previous.Title && job.Description == previous.Description && job.Location == previous.Location && job.SalaryRaw == previous.SalaryRaw && job.WorkArrangement == previous.WorkArrangement {
			return previous, "unchanged", nil
		}
		m.jobs[job.URL] = job
		return job, "changed", nil
	}
	job.ID = "mock-" + strconv.Itoa(len(m.jobs)+1)
	m.jobs[job.URL] = job
	return job, "new", nil
}

func (m *MockJobProvider) NewURLs(_ context.Context, urls []string) ([]string, error) {
	if m.NewURLsErr != nil {
		return nil, m.NewURLsErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, u := range urls {
		if _, exists := m.jobs[u]; !exists {
			out = append(out, u)
		}
	}
	return out, nil
}

func (m *MockJobProvider) List(_ context.Context, _ string) ([]dto.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out, nil
}

func (m *MockJobProvider) Page(ctx context.Context, userID string, options JobPageOptions) (JobPage, error) {
	jobs, err := m.List(ctx, userID)
	if err != nil {
		return JobPage{}, err
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].ScrapedAt.Equal(jobs[j].ScrapedAt) {
			return jobs[i].ID > jobs[j].ID
		}
		return jobs[i].ScrapedAt.After(jobs[j].ScrapedAt)
	})
	page := JobPage{Items: make([]dto.Job, 0)}
	for _, job := range jobs {
		if options.CompanyID != "" && job.CompanyID != options.CompanyID {
			continue
		}
		if !options.CursorTime.IsZero() && (job.ScrapedAt.After(options.CursorTime) || (job.ScrapedAt.Equal(options.CursorTime) && job.ID >= options.CursorID)) {
			continue
		}
		page.Items = append(page.Items, job)
		if len(page.Items) == int(options.Limit) {
			break
		}
	}
	return page, nil
}

func (m *MockJobProvider) GetJob(_ context.Context, jobID, _ string) (dto.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if j.ID == jobID {
			return j, nil
		}
	}
	return dto.Job{}, ErrNotFound
}

func (m *MockJobProvider) ListSince(_ context.Context, since time.Time) ([]dto.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []dto.Job
	for _, j := range m.jobs {
		if j.ScrapedAt.After(since) {
			out = append(out, j)
		}
	}
	return out, nil
}

func (m *MockJobProvider) Jobs() []dto.Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]dto.Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out
}
