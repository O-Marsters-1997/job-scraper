package ingest_test

import (
	"context"
	"errors"
	"sync"
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
	mu    sync.Mutex
	calls []struct {
		job    dto.Job
		userID string
	}
}

func (s *stubScorer) ScoreAndSave(_ context.Context, job dto.Job, userID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, struct {
		job    dto.Job
		userID string
	}{job, userID})
	return 0
}

type stubUserLister struct {
	users []string
	err   error
}

func (l *stubUserLister) ListUsersWithProvider(_ context.Context, _ string) ([]string, error) {
	return l.users, l.err
}

type stubCredGetter struct {
	key string
	err error
}

func (c *stubCredGetter) Get(_ context.Context, _, _ string) (string, error) {
	return c.key, c.err
}

type stubNotifier struct {
	calls []struct {
		job   dto.Job
		email string
	}
}

func (n *stubNotifier) NotifyNewJob(_ context.Context, email string, job dto.Job, _ int) {
	n.calls = append(n.calls, struct {
		job   dto.Job
		email string
	}{job, email})
}

type stubEmailGetter struct {
	email string
	err   error
}

func (e *stubEmailGetter) GetUserEmail(_ context.Context, _ string) (string, error) {
	return e.email, e.err
}

func (e *stubEmailGetter) GetProfile(_ context.Context, _ string) (dto.Profile, error) {
	return dto.Profile{Email: e.email}, e.err
}

func (e *stubEmailGetter) UpdateEmail(_ context.Context, _, _ string) (dto.Profile, error) {
	return dto.Profile{}, nil
}

// oneUserCfg returns a Config wired with a single user and a shared stubScorer.
func oneUserCfg(db ingest.Saver, sc *stubScorer, notifier ingest.Notifier) ingest.Config {
	return ingest.Config{
		DB:       db,
		Provider: "anthropic",
		Users:    &stubUserLister{users: []string{"user-1"}},
		Creds:    &stubCredGetter{key: "key-1"},
		ScorerFor: func(_ string) ingest.Scorer {
			return sc
		},
		Emails:   &stubEmailGetter{email: "user@example.com"},
		Notifier: notifier,
	}
}

func TestIngest(t *testing.T) {
	t.Parallel()

	errSave := errors.New("db down")

	tests := []struct {
		name         string
		jobs         []dto.Job
		saveErr      error
		noScoring    bool
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
			name:        "no scoring config and nil notifier do not panic",
			jobs:        []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}},
			noScoring:   true,
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

			var cfg ingest.Config
			if tt.noScoring {
				cfg = ingest.Config{DB: db}
			} else {
				cfg = oneUserCfg(db, sc, nc)
				if tt.nilNotifier {
					cfg.Notifier = nil
				}
			}

			err := ingest.New(cfg).Ingest(context.Background(), tt.jobs)

			if tt.wantErr {
				if err == nil {
					t.Error("Ingest: expected error, got nil")
				}
				if len(sc.calls) != 0 {
					t.Errorf("scorer called %d times after save error; want 0", len(sc.calls))
				}
				if len(nc.calls) != 0 {
					t.Errorf("notifier called %d times after save error; want 0", len(nc.calls))
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
			if len(sc.calls) != tt.wantScored {
				t.Errorf("scorer called %d times; want %d", len(sc.calls), tt.wantScored)
			}
			if len(nc.calls) != tt.wantNotified {
				t.Errorf("notifier called %d times; want %d", len(nc.calls), tt.wantNotified)
			}
		})
	}
}

func TestIngest_TwoUserFanOut(t *testing.T) {
	t.Parallel()

	db := &stubSaver{}
	sc := &stubScorer{}

	cfg := ingest.Config{
		DB:       db,
		Provider: "anthropic",
		Users:    &stubUserLister{users: []string{"alice", "bob"}},
		Creds:    &stubCredGetter{key: "key-x"},
		ScorerFor: func(_ string) ingest.Scorer {
			return sc
		},
	}

	jobs := []dto.Job{
		{Title: "Engineer", URL: "https://example.com/1"},
		{Title: "Manager", URL: "https://example.com/2"},
	}
	if err := ingest.New(cfg).Ingest(context.Background(), jobs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 jobs × 2 users = 4 ScoreAndSave calls
	if len(sc.calls) != 4 {
		t.Errorf("want 4 scorer calls (2 jobs × 2 users), got %d", len(sc.calls))
	}

	// Each user should appear exactly twice (once per job)
	counts := map[string]int{}
	for _, c := range sc.calls {
		counts[c.userID]++
	}
	for _, uid := range []string{"alice", "bob"} {
		if counts[uid] != 2 {
			t.Errorf("user %q scored %d times; want 2", uid, counts[uid])
		}
	}
}

func TestIngest_CredentialErrorSkipsUser(t *testing.T) {
	t.Parallel()

	db := &stubSaver{}
	sc := &stubScorer{}

	credErr := errors.New("decryption failed")
	creds := &multiCredGetter{
		keys: map[string]string{"alice": "key-a"},
		errs: map[string]error{"bob": credErr},
	}

	cfg := ingest.Config{
		DB:       db,
		Provider: "anthropic",
		Users:    &stubUserLister{users: []string{"alice", "bob"}},
		Creds:    creds,
		ScorerFor: func(_ string) ingest.Scorer {
			return sc
		},
	}

	if err := ingest.New(cfg).Ingest(context.Background(), []dto.Job{
		{Title: "Engineer", URL: "https://example.com/1"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// only alice succeeds; bob is skipped
	if len(sc.calls) != 1 {
		t.Errorf("want 1 scorer call (alice only), got %d", len(sc.calls))
	}
	if sc.calls[0].userID != "alice" {
		t.Errorf("expected alice to be scored, got %q", sc.calls[0].userID)
	}
}

// multiCredGetter returns per-user keys or errors.
type multiCredGetter struct {
	keys map[string]string
	errs map[string]error
}

func (m *multiCredGetter) Get(_ context.Context, userID, _ string) (string, error) {
	if err, ok := m.errs[userID]; ok {
		return "", err
	}
	return m.keys[userID], nil
}

func TestProviderForModel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		modelID string
		want    string
	}{
		{"claude-haiku-4-5-20251001", "anthropic"},
		{"claude-sonnet-4-6", "anthropic"},
		{"claude-3-opus", "anthropic"},
		{"gpt-4", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ingest.ProviderForModel(tt.modelID); got != tt.want {
			t.Errorf("ProviderForModel(%q) = %q; want %q", tt.modelID, got, tt.want)
		}
	}
}
