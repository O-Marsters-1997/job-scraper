package sourcetargets_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/runwindow"
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
	target, err := st.CreateSourceTarget(t.Context(), userID, wisTarget.Source, wisTarget.Value, enabled, nil, dto.RunWindow{}, nil)
	if err != nil {
		t.Fatalf("CreateSourceTarget() err = %v", err)
	}
	return target
}

func withInterval(minutes int) func(*dto.RunWindow) {
	return func(w *dto.RunWindow) { w.IntervalMinutes = &minutes }
}

func runWindowInput(edit func(*dto.RunWindow)) dto.RunWindowInput {
	w := runwindow.Default()
	edit(&w)
	return dto.RunWindowInput{Set: true, Window: &w}
}

func assertInDefaultWindow(t *testing.T, at time.Time) {
	t.Helper()
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatalf("LoadLocation() err = %v", err)
	}
	local := at.In(london)
	minutes := local.Hour()*60 + local.Minute()
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday || minutes < 8*60 || minutes >= 18*60 {
		t.Errorf("next run %v is outside Mon-Fri 08:00-18:00 Europe/London", local)
	}
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

	t.Run("defaults to the hourly weekday window", func(t *testing.T) {
		svc, _, _ := newService(t)
		before := time.Now()
		target, err := svc.Create(t.Context(), userID, wisTarget)
		if err != nil {
			t.Fatalf("Create() err = %v", err)
		}
		if diff := cmp.Diff(runwindow.Default(), target.RunWindow); diff != "" {
			t.Errorf("Create().RunWindow mismatch (-want +got):\n%s", diff)
		}
		if target.NextRunAt == nil || !target.NextRunAt.After(before) {
			t.Fatalf("Create().NextRunAt = %v, want after %v", target.NextRunAt, before)
		}
		assertInDefaultWindow(t, *target.NextRunAt)
	})

	t.Run("a null run window creates a manual target", func(t *testing.T) {
		svc, _, _ := newService(t)
		in := wisTarget
		in.RunWindow = dto.RunWindowInput{Set: true}
		target, err := svc.Create(t.Context(), userID, in)
		if err != nil {
			t.Fatalf("Create() err = %v", err)
		}
		if target.RunWindow.IntervalMinutes != nil || target.NextRunAt != nil {
			t.Errorf("Create(null) interval = %v, next run = %v, want both nil", target.RunWindow.IntervalMinutes, target.NextRunAt)
		}
	})

	t.Run("rejects an invalid run window", func(t *testing.T) {
		svc, _, _ := newService(t)
		in := wisTarget
		in.RunWindow = runWindowInput(func(w *dto.RunWindow) { w.Timezone = "Mars/Olympus" })
		if _, err := svc.Create(t.Context(), userID, in); !apperr.IsKind(err, apperr.KindInvalid) {
			t.Fatalf("Create() err = %v, want kind %v", err, apperr.KindInvalid)
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
			name        string
			in          dto.UpdateSourceTargetInput
			wantEnabled bool
		}{
			{name: "enabled flag", in: dto.UpdateSourceTargetInput{Enabled: new(false)}, wantEnabled: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, st, _ := newService(t)
				tt.in.ID = createTarget(t, st, true).ID
				got, err := svc.Update(t.Context(), userID, tt.in)
				if err != nil {
					t.Fatalf("Update() err = %v", err)
				}
				if got.Enabled != tt.wantEnabled {
					t.Errorf("Update() enabled = %t, want %t", got.Enabled, tt.wantEnabled)
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
			{name: "interval below 60", in: dto.UpdateSourceTargetInput{RunWindow: runWindowInput(withInterval(30))}, wantKind: apperr.KindInvalid},
			{name: "unknown timezone", in: dto.UpdateSourceTargetInput{RunWindow: runWindowInput(func(w *dto.RunWindow) { w.Timezone = "Mars/Olympus" })}, wantKind: apperr.KindInvalid},
			{name: "no weekdays", in: dto.UpdateSourceTargetInput{RunWindow: runWindowInput(func(w *dto.RunWindow) { w.Weekdays = nil })}, wantKind: apperr.KindInvalid},
			{name: "inverted window", in: dto.UpdateSourceTargetInput{RunWindow: runWindowInput(func(w *dto.RunWindow) { w.Start, w.End = "18:00", "08:00" })}, wantKind: apperr.KindInvalid},
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

	t.Run("a run window replaces the schedule and resets the next run", func(t *testing.T) {
		svc, st, _ := newService(t)
		created := createTarget(t, st, true)

		before := time.Now()
		got, err := svc.Update(t.Context(), userID, dto.UpdateSourceTargetInput{ID: created.ID, RunWindow: runWindowInput(withInterval(180))})
		if err != nil {
			t.Fatalf("Update() err = %v", err)
		}
		if iv := got.RunWindow.IntervalMinutes; iv == nil || *iv != 180 {
			t.Errorf("Update().RunWindow.IntervalMinutes = %v, want 180", iv)
		}
		if got.NextRunAt == nil || !got.NextRunAt.After(before) {
			t.Fatalf("Update().NextRunAt = %v, want after %v", got.NextRunAt, before)
		}
		assertInDefaultWindow(t, *got.NextRunAt)
	})

	t.Run("enabling reschedules the next run", func(t *testing.T) {
		svc, _, _ := newService(t)
		in := wisTarget
		in.Enabled = new(false)
		created, err := svc.Create(t.Context(), userID, in)
		if err != nil {
			t.Fatalf("Create() err = %v", err)
		}
		if created.NextRunAt != nil {
			t.Fatalf("Create(disabled).NextRunAt = %v, want nil", created.NextRunAt)
		}

		got, err := svc.Update(t.Context(), userID, dto.UpdateSourceTargetInput{ID: created.ID, Enabled: new(true)})
		if err != nil {
			t.Fatalf("Update() err = %v", err)
		}
		if got.NextRunAt == nil {
			t.Fatal("Update(enable).NextRunAt = nil, want a scheduled run")
		}
		assertInDefaultWindow(t, *got.NextRunAt)
	})

	t.Run("a null run window makes the target manual", func(t *testing.T) {
		svc, _, _ := newService(t)
		created, err := svc.Create(t.Context(), userID, wisTarget)
		if err != nil {
			t.Fatalf("Create() err = %v", err)
		}

		got, err := svc.Update(t.Context(), userID, dto.UpdateSourceTargetInput{ID: created.ID, RunWindow: dto.RunWindowInput{Set: true}})
		if err != nil {
			t.Fatalf("Update() err = %v", err)
		}
		if got.RunWindow.IntervalMinutes != nil || got.NextRunAt != nil {
			t.Errorf("Update(null) interval = %v, next run = %v, want both nil", got.RunWindow.IntervalMinutes, got.NextRunAt)
		}
		if got.RunWindow.Timezone != "Europe/London" || len(got.RunWindow.Weekdays) != 5 {
			t.Errorf("Update(null).RunWindow = %+v, want the stored window kept", got.RunWindow)
		}
	})

	t.Run("a null run window on another user's target is not found", func(t *testing.T) {
		svc, st, _ := newService(t)
		created := createTarget(t, st, true)
		_, err := svc.Update(t.Context(), "user-2", dto.UpdateSourceTargetInput{ID: created.ID, RunWindow: dto.RunWindowInput{Set: true}})
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("Update() err = %v, want kind %v", err, apperr.KindNotFound)
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

	t.Run("a rerun resets the next run", func(t *testing.T) {
		svc, _, _ := newService(t)
		created, err := svc.Create(t.Context(), userID, dto.CreateSourceTargetInput{Source: "wis", Value: "rerun", RunWindow: runWindowInput(withInterval(60))})
		if err != nil {
			t.Fatalf("Create() err = %v", err)
		}
		if _, err := svc.TransitionSourceTargetRun(t.Context(), created.ID, created.RunID, "succeeded", ""); err != nil {
			t.Fatalf("TransitionSourceTargetRun() err = %v", err)
		}

		before := time.Now()
		rerun, err := svc.Scrape(t.Context(), userID, created.ID)
		if err != nil {
			t.Fatalf("Scrape() err = %v", err)
		}
		if rerun.NextRunAt == nil || !rerun.NextRunAt.After(before) {
			t.Fatalf("Scrape().NextRunAt = %v, want after %v", rerun.NextRunAt, before)
		}
		assertInDefaultWindow(t, *rerun.NextRunAt)
	})

	t.Run("retries a failed run", func(t *testing.T) {
		svc, st, q := newService(t)
		created := createTarget(t, st, true)
		started, err := st.StartSourceTargetRun(t.Context(), created.ID, nil)
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

func TestDisableSource(t *testing.T) {
	t.Run("counts the targets it disabled by source and reason", func(t *testing.T) {
		svc, st, _ := newService(t)
		createTarget(t, st, true)
		if _, err := st.CreateSourceTarget(t.Context(), userID, "wis", "designer", true, nil, dto.RunWindow{}, nil); err != nil {
			t.Fatalf("CreateSourceTarget() err = %v", err)
		}
		counter := sourcetargets.TargetsDisabled.WithLabelValues("wis", "wis key rejected")
		before := testutil.ToFloat64(counter)

		n, err := svc.DisableSource(t.Context(), "wis", "wis key rejected")
		if err != nil || n != 2 {
			t.Fatalf("DisableSource() = %d, %v, want 2, nil", n, err)
		}
		if got := testutil.ToFloat64(counter) - before; got != 2 {
			t.Errorf("jobscraper_source_targets_disabled_total rose by %v, want 2", got)
		}

		if _, err := svc.DisableSource(t.Context(), "wis", "wis key rejected"); err != nil {
			t.Fatalf("second DisableSource() err = %v", err)
		}
		if got := testutil.ToFloat64(counter) - before; got != 2 {
			t.Errorf("counter rose by %v after a no-op disable, want 2", got)
		}
	})
}
