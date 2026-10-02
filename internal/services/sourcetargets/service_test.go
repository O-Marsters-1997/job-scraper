package sourcetargets_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

const userID = "user-1"

var wisTarget = dto.CreateSourceTargetInput{Source: "wis", Value: "engineer"}

type fakeSearchConfigReader struct{}

func (fakeSearchConfigReader) SearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, data.ErrNotFound
}

type failingCandidateList struct {
	*jobsearchtest.FakeStore
	err error
}

func (f failingCandidateList) ListForUser(context.Context, string, string, int) ([]sourcetargets.Candidate, error) {
	return nil, f.err
}

type verifiedBoards []dto.CardBoard

func (v verifiedBoards) VerifiedBoardsBySlug(_ context.Context, slugs []string) ([]dto.CardBoard, error) {
	return slices.DeleteFunc(slices.Clone(v), func(b dto.CardBoard) bool {
		return !slices.Contains(slugs, b.CompanySlug)
	}), nil
}

func serviceOver(targets sourcetargets.Store, q sourcetargets.QueuePublisher) *sourcetargets.Service {
	return sourcetargets.New(targets, fakeSearchConfigReader{}, q, verifiedBoards(nil))
}

func newService(t *testing.T) (*sourcetargets.Service, *jobsearchtest.FakeStore, *queuetest.Recorder) {
	t.Helper()
	st := jobsearchtest.NewFakeStore()
	q := queuetest.NewRecorder()
	return serviceOver(st, q), st, q
}

func createTarget(t *testing.T, st *jobsearchtest.FakeStore, enabled bool) dto.SourceTarget {
	t.Helper()
	target, err := st.CreateSourceTarget(t.Context(), userID, wisTarget.Source, wisTarget.Value, enabled, nil)
	if err != nil {
		t.Fatalf("CreateSourceTarget() err = %v", err)
	}
	return target
}

func TestCreate(t *testing.T) {
	t.Run("rejects bad input", func(t *testing.T) {
		tests := []struct {
			name string
			in   dto.CreateSourceTargetInput
		}{
			{"missing source and value", dto.CreateSourceTargetInput{}},
			{"unsupported source", dto.CreateSourceTargetInput{Source: "unknown-ats", Value: "x"}},
			{"ATS source", dto.CreateSourceTargetInput{Source: "greenhouse", Value: "acme"}},
		}
		svc, _, _ := newService(t)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := svc.Create(t.Context(), userID, tt.in)
				if !apperr.IsKind(err, apperr.KindInvalid) {
					t.Fatalf("Create(%+v) err = %v, want kind %v", tt.in, err, apperr.KindInvalid)
				}
			})
		}
	})

	t.Run("a discovery source queues one run", func(t *testing.T) {
		svc, _, q := newService(t)
		target, err := svc.Create(t.Context(), userID, wisTarget)
		if err != nil {
			t.Fatalf("Create() err = %v", err)
		}
		if target.RunStatus != "queued" {
			t.Errorf("run status = %q, want queued", target.RunStatus)
		}
		if got := q.Tasks(); len(got) != 1 || got[0].Source != "wis" {
			t.Errorf("queued tasks = %+v, want one WIS search", got)
		}
	})

	t.Run("linkedin recency", func(t *testing.T) {
		tests := []struct {
			name    string
			filters map[string]string
			want    string
		}{
			{"defaults to past week", nil, "r604800"},
			{"keeps an explicit value", map[string]string{"recency": "r86400"}, "r86400"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, _, _ := newService(t)
				target, err := svc.Create(t.Context(), userID, dto.CreateSourceTargetInput{Source: "linkedin", Value: "engineer", Filters: tt.filters})
				if err != nil {
					t.Fatalf("Create() err = %v", err)
				}
				if got := target.Filters["recency"]; got != tt.want {
					t.Errorf("Create().Filters[recency] = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("keeps a recoverable run after a queue failure", func(t *testing.T) {
		svc := serviceOver(jobsearchtest.NewFakeStore(), queuetest.PublishFails(errors.New("queue unavailable")))
		target, err := svc.Create(t.Context(), userID, wisTarget)
		if err != nil {
			t.Fatalf("Create() err = %v", err)
		}
		if target.RunStatus != "queued" || target.RunID == "" {
			t.Errorf("target = %+v, want a queued run", target)
		}
	})

	t.Run("a duplicate conflicts", func(t *testing.T) {
		tests := []struct {
			name        string
			first       dto.CreateSourceTargetInput
			second      dto.CreateSourceTargetInput
			wantStoredV string
		}{
			{
				name:   "same value",
				first:  dto.CreateSourceTargetInput{Source: "wis", Value: "engineer", Enabled: new(false)},
				second: dto.CreateSourceTargetInput{Source: "wis", Value: "engineer", Enabled: new(false)},
			},
			{
				name:        "indeed urls differing in tracking params",
				first:       dto.CreateSourceTargetInput{Source: "indeed", Value: "https://www.indeed.com/jobs?q=golang&l=London&vjk=abc"},
				second:      dto.CreateSourceTargetInput{Source: "indeed", Value: "https://www.indeed.com/jobs?l=London&q=golang&from=searchOnDesktopSerp&start=10"},
				wantStoredV: "golang",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, _, _ := newService(t)
				target, err := svc.Create(t.Context(), userID, tt.first)
				if err != nil {
					t.Fatalf("first Create() err = %v", err)
				}
				if tt.wantStoredV != "" && target.Value != tt.wantStoredV {
					t.Errorf("stored value = %q, want %q", target.Value, tt.wantStoredV)
				}
				if _, err := svc.Create(t.Context(), userID, tt.second); !apperr.IsKind(err, apperr.KindConflict) {
					t.Fatalf("second Create() err = %v, want kind %v", err, apperr.KindConflict)
				}
			})
		}
	})
}

func TestList(t *testing.T) {
	svc, _, _ := newService(t)
	in := wisTarget
	in.Filters = map[string]string{"loc": "86383"}
	in.Enabled = new(false)
	if _, err := svc.Create(t.Context(), userID, in); err != nil {
		t.Fatalf("Create() err = %v", err)
	}

	got, err := svc.List(t.Context(), userID)
	if err != nil {
		t.Fatalf("List() err = %v", err)
	}
	if len(got) != 1 || got[0].URL != "https://workinstartups.com/search?loc=86383&q=engineer" {
		t.Errorf("List() = %+v, want one target with the board URL", got)
	}
}

func TestUpdate(t *testing.T) {
	t.Run("applies the change", func(t *testing.T) {
		tests := []struct {
			name         string
			in           dto.UpdateSourceTargetInput
			wantEnabled  bool
			wantInterval int
		}{
			{name: "enabled flag", in: dto.UpdateSourceTargetInput{Enabled: new(false)}, wantEnabled: false},
			{name: "check interval", in: dto.UpdateSourceTargetInput{CheckIntervalMinutes: new(60)}, wantEnabled: true, wantInterval: 60},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, st, _ := newService(t)
				tt.in.ID = createTarget(t, st, true).ID
				got, err := svc.Update(t.Context(), userID, tt.in)
				if err != nil {
					t.Fatalf("Update() err = %v", err)
				}
				if got.Enabled != tt.wantEnabled || got.CheckIntervalMinutes != tt.wantInterval {
					t.Errorf("Update() enabled = %t, interval = %d, want %t, %d", got.Enabled, got.CheckIntervalMinutes, tt.wantEnabled, tt.wantInterval)
				}
			})
		}
	})

	t.Run("rejects bad input", func(t *testing.T) {
		tests := []struct {
			name     string
			in       dto.UpdateSourceTargetInput
			missing  bool
			wantKind apperr.Kind
		}{
			{name: "interval below 60", in: dto.UpdateSourceTargetInput{CheckIntervalMinutes: new(30)}, wantKind: apperr.KindInvalid},
			{name: "empty body", wantKind: apperr.KindInvalid},
			{name: "unknown target", in: dto.UpdateSourceTargetInput{Enabled: new(false)}, missing: true, wantKind: apperr.KindNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, st, _ := newService(t)
				tt.in.ID = createTarget(t, st, true).ID
				if tt.missing {
					tt.in.ID = "missing-id"
				}
				_, err := svc.Update(t.Context(), userID, tt.in)
				if !apperr.IsKind(err, tt.wantKind) {
					t.Fatalf("Update() err = %v, want kind %v", err, tt.wantKind)
				}
			})
		}
	})

	t.Run("enabling a discovery target reconsiders its candidates", func(t *testing.T) {
		svc, st, q := newService(t)
		created := createTarget(t, st, false)
		card := dto.Job{URL: "https://example.com/1", Title: "Engineer"}
		if _, err := st.SaveCards(t.Context(), created, []dto.Job{card}); err != nil {
			t.Fatalf("SaveCards() err = %v", err)
		}

		if _, err := svc.Update(t.Context(), userID, dto.UpdateSourceTargetInput{ID: created.ID, Enabled: new(true)}); err != nil {
			t.Fatalf("Update() err = %v", err)
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != card.URL {
			t.Errorf("queued after enabling = %+v, want one job for %s", jobs, card.URL)
		}
	})

	t.Run("a reconsideration failure is unavailable", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		created := createTarget(t, st, false)
		svc := serviceOver(failingCandidateList{FakeStore: st, err: errors.New("boom")}, queuetest.NewRecorder())

		_, err := svc.Update(t.Context(), userID, dto.UpdateSourceTargetInput{ID: created.ID, Enabled: new(true)})
		if !apperr.IsKind(err, apperr.KindUnavailable) {
			t.Fatalf("Update() err = %v, want kind %v", err, apperr.KindUnavailable)
		}
	})
}

func TestScrape(t *testing.T) {
	t.Run("queues a run once, for the owner only", func(t *testing.T) {
		svc, st, _ := newService(t)
		created := createTarget(t, st, true)

		if _, err := svc.Scrape(t.Context(), "user-2", created.ID); !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("Scrape(other user) err = %v, want kind %v", err, apperr.KindNotFound)
		}
		target, err := svc.Scrape(t.Context(), userID, created.ID)
		if err != nil {
			t.Fatalf("Scrape() err = %v", err)
		}
		if target.RunStatus != "queued" {
			t.Errorf("run status = %q, want queued", target.RunStatus)
		}
		if _, err := svc.Scrape(t.Context(), userID, created.ID); !apperr.IsKind(err, apperr.KindConflict) {
			t.Errorf("second Scrape() err = %v, want kind %v", err, apperr.KindConflict)
		}
	})

	t.Run("retries a failed run", func(t *testing.T) {
		svc, st, q := newService(t)
		created := createTarget(t, st, true)
		started, err := st.StartSourceTargetRun(t.Context(), created.ID)
		if err != nil {
			t.Fatalf("StartSourceTargetRun() err = %v", err)
		}
		if _, err := st.TransitionSourceTargetRun(t.Context(), created.ID, started.RunID, "failed", "previous run failed"); err != nil {
			t.Fatalf("TransitionSourceTargetRun() err = %v", err)
		}

		target, err := svc.Scrape(t.Context(), userID, created.ID)
		if err != nil {
			t.Fatalf("Scrape() err = %v", err)
		}
		if target.RunStatus != "queued" || target.LastRunError != "" {
			t.Errorf("run state after retry = %+v, want queued with no error", target)
		}
		if got := len(q.Tasks()); got != 1 {
			t.Errorf("queued tasks = %d, want 1", got)
		}
	})
}
