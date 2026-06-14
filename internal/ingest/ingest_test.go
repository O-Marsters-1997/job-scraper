package ingest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
)

type stubSaver struct {
	calls [][]dto.Job
	err   error
}

func (s *stubSaver) Save(_ context.Context, jobs []dto.Job) error {
	if s.err != nil {
		return s.err
	}
	s.calls = append(s.calls, jobs)
	return nil
}

type stubScorer struct {
	jobs []dto.Job
}

func (s *stubScorer) ScoreAndSave(_ context.Context, job dto.Job) int {
	s.jobs = append(s.jobs, job)
	return 0
}

type stubNotifier struct {
	jobs []dto.Job
}

func (n *stubNotifier) NotifyNewJob(_ context.Context, job dto.Job, _ int) {
	n.jobs = append(n.jobs, job)
}

func TestIngest(t *testing.T) {
	t.Parallel()

	errSave := errors.New("db down")

	tests := []struct {
		name         string
		jobs         []dto.Job
		saveErr      error
		nilScorer    bool
		nilNotifier  bool
		wantErr      bool
		wantSaved    int
		wantScored   int
		wantNotified int
	}{
		{
			name: "valid batch is saved, scored, and notified",
			jobs: []dto.Job{
				{Title: "Engineer", URL: "https://example.com/1"},
				{Title: "Manager", URL: "https://example.com/2"},
			},
			wantSaved:    2,
			wantScored:   2,
			wantNotified: 2,
		},
		{
			name: "jobs missing title or url are filtered out",
			jobs: []dto.Job{
				{Title: "", URL: "https://example.com/1"},
				{Title: "Engineer", URL: ""},
				{Title: "Valid", URL: "https://example.com/3"},
			},
			wantSaved:    1,
			wantScored:   1,
			wantNotified: 1,
		},
		{
			name:      "all invalid - nothing saved",
			jobs:      []dto.Job{{Title: "", URL: ""}},
			wantSaved: 0,
		},
		{
			name:      "nil input - nothing saved",
			wantSaved: 0,
		},
		{
			name:    "save error propagates; scorer and notifier not called",
			jobs:    []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}},
			saveErr: errSave,
			wantErr: true,
		},
		{
			name:        "nil scorer and notifier do not panic",
			jobs:        []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}},
			nilScorer:   true,
			nilNotifier: true,
			wantSaved:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := &stubSaver{err: tt.saveErr}
			sc := &stubScorer{}
			nc := &stubNotifier{}

			var scorer ingest.Scorer
			if !tt.nilScorer {
				scorer = sc
			}
			var notifier ingest.Notifier
			if !tt.nilNotifier {
				notifier = nc
			}

			err := ingest.New(db, scorer, notifier).Ingest(context.Background(), tt.jobs)

			if tt.wantErr {
				if err == nil {
					t.Error("Ingest: expected error, got nil")
				}
				if len(sc.jobs) != 0 {
					t.Errorf("scorer called %d times after save error; want 0", len(sc.jobs))
				}
				if len(nc.jobs) != 0 {
					t.Errorf("notifier called %d times after save error; want 0", len(nc.jobs))
				}
				return
			}

			if err != nil {
				t.Fatalf("Ingest: unexpected error: %v", err)
			}

			var saved int
			if len(db.calls) > 0 {
				saved = len(db.calls[0])
			}
			if saved != tt.wantSaved {
				t.Errorf("saved %d jobs; want %d", saved, tt.wantSaved)
			}
			if len(sc.jobs) != tt.wantScored {
				t.Errorf("scorer called %d times; want %d", len(sc.jobs), tt.wantScored)
			}
			if len(nc.jobs) != tt.wantNotified {
				t.Errorf("notifier called %d times; want %d", len(nc.jobs), tt.wantNotified)
			}
		})
	}
}
