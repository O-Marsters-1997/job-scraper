package jobsearch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

type failingPublisher struct {
	jobsearchtest.NoopQueue
	err error
}

func (f failingPublisher) Publish(context.Context, queue.Task) error { return f.err }

func newCompanyService(q jobsearch.QueuePublisher) (*jobsearch.Service, *jobsearchtest.FakeStore) {
	st := jobsearchtest.NewFakeStore()
	return jobsearch.NewService(st, st, q), st
}

func listTargets(t *testing.T, st *jobsearchtest.FakeStore) []dto.SourceTarget {
	t.Helper()
	targets, err := st.ListSourceTargetsByUser(t.Context(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	return targets
}

func trackingFor(t *testing.T, st *jobsearchtest.FakeStore, userID, companyID string) (dto.Company, bool) {
	t.Helper()
	companies, err := st.ListCompaniesForUser(t.Context(), userID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range companies {
		if c.ID == companyID {
			return c, true
		}
	}
	return dto.Company{}, false
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name  string
		in    dto.CreateCompanyInput
		check func(t *testing.T, company dto.Company, err error, companyStore *jobsearchtest.FakeStore)
	}{
		{
			name:  "rejects missing url",
			in:    dto.CreateCompanyInput{},
			check: wantCreateErr(apperr.KindInvalid),
		},
		{
			name:  "rejects an unresolvable url",
			in:    dto.CreateCompanyInput{URL: "https://example.com/careers"},
			check: wantCreateErr(apperr.KindUnprocessable),
		},
		{
			name:  "resolves and tracks a valid board url by default",
			in:    dto.CreateCompanyInput{URL: "https://boards.greenhouse.io/acmecorp"},
			check: wantCreated("acmecorp", true),
		},
		{
			name:  "does not track when track is explicitly false",
			in:    dto.CreateCompanyInput{URL: "https://boards.greenhouse.io/acmecorp", Track: boolPtr(false)},
			check: wantCreated("acmecorp", false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, companyStore := newCompanyService(queuetest.NewRecorder())

			company, err := svc.CreateCompany(t.Context(), "user-1", tt.in)
			tt.check(t, company, err, companyStore)
		})
	}
}

func wantCreateErr(wantKind apperr.Kind) func(t *testing.T, _ dto.Company, err error, _ *jobsearchtest.FakeStore) {
	return func(t *testing.T, _ dto.Company, err error, _ *jobsearchtest.FakeStore) {
		t.Helper()
		assertKind(t, err, wantKind)
	}
}

func wantCreated(wantSlug string, wantTrack bool) func(t *testing.T, company dto.Company, err error, companyStore *jobsearchtest.FakeStore) {
	return func(t *testing.T, company dto.Company, err error, companyStore *jobsearchtest.FakeStore) {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if company.Slug != wantSlug {
			t.Errorf("slug = %q, want %q", company.Slug, wantSlug)
		}
		tracked, ok := trackingFor(t, companyStore, "user-1", company.ID)
		if got := ok && tracked.Tracked; got != wantTrack {
			t.Errorf("tracked = %v, want %v", got, wantTrack)
		}
	}
}

func TestSetTracking(t *testing.T) {
	t.Run("rejects a missing enabled field", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})

		_, err := svc.SetCompanyTracking(t.Context(), "user-1", dto.SetCompanyTrackingInput{CompanyID: company.ID})

		assertKind(t, err, apperr.KindInvalid)
	})

	t.Run("rejects an interval below 60 minutes", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})

		_, err := svc.SetCompanyTracking(t.Context(), "user-1", dto.SetCompanyTrackingInput{
			CompanyID: company.ID, Enabled: boolPtr(true), CheckIntervalMinutes: intPtr(30),
		})

		assertKind(t, err, apperr.KindInvalid)
	})

	t.Run("returns not found for an unknown company", func(t *testing.T) {
		svc, _ := newCompanyService(queuetest.NewRecorder())

		_, err := svc.SetCompanyTracking(t.Context(), "user-1", dto.SetCompanyTrackingInput{CompanyID: "missing", Enabled: boolPtr(true)})

		assertKind(t, err, apperr.KindNotFound)
	})

	t.Run("enables tracking and syncs the legacy source target for an ATS company", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"})

		tracking, err := svc.SetCompanyTracking(t.Context(), "user-1", dto.SetCompanyTrackingInput{
			CompanyID: company.ID, Enabled: boolPtr(true), CheckIntervalMinutes: intPtr(180),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !tracking.Enabled || tracking.CheckIntervalMinutes != 180 {
			t.Errorf("tracking = %+v", tracking)
		}
		legacy := listTargets(t, companyStore)
		if len(legacy) != 1 || legacy[0].CheckIntervalMinutes != 180 {
			t.Errorf("legacy target = %+v", legacy)
		}
	})

	t.Run("tracks a company without a board and skips the legacy sync", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})

		_, err := svc.SetCompanyTracking(t.Context(), "user-1", dto.SetCompanyTrackingInput{
			CompanyID: company.ID, Enabled: boolPtr(true), CheckIntervalMinutes: intPtr(180),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if legacy := listTargets(t, companyStore); len(legacy) != 0 {
			t.Errorf("want no legacy target, got %+v", legacy)
		}
	})

	t.Run("preserves the existing frequency when no interval is given", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		if _, err := svc.SetCompanyTracking(t.Context(), "user-1", dto.SetCompanyTrackingInput{CompanyID: company.ID, Enabled: boolPtr(true), CheckIntervalMinutes: intPtr(180)}); err != nil {
			t.Fatal(err)
		}

		tracking, err := svc.SetCompanyTracking(t.Context(), "user-1", dto.SetCompanyTrackingInput{CompanyID: company.ID, Enabled: boolPtr(false)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tracking.Enabled || tracking.CheckIntervalMinutes != 180 {
			t.Errorf("tracking = %+v", tracking)
		}
	})
}

func TestListBoards(t *testing.T) {
	t.Run("returns not found for an unknown company", func(t *testing.T) {
		svc, _ := newCompanyService(queuetest.NewRecorder())

		_, err := svc.ListCompanyBoards(t.Context(), "user-1", "missing")

		assertKind(t, err, apperr.KindNotFound)
	})

	t.Run("lists boards linked to the company", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		_, err := companyStore.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "acme")
		if err != nil {
			t.Fatal(err)
		}

		boards, err := svc.ListCompanyBoards(t.Context(), "user-1", company.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(boards) != 1 {
			t.Errorf("want 1 board, got %d", len(boards))
		}
	})
}

func TestAddBoard(t *testing.T) {
	t.Run("returns not found for an unknown company", func(t *testing.T) {
		svc, _ := newCompanyService(queuetest.NewRecorder())

		_, err := svc.AddCompanyBoard(t.Context(), "user-1", dto.AddCompanyBoardInput{CompanyID: "missing", URL: "https://boards.greenhouse.io/acme"})

		assertKind(t, err, apperr.KindNotFound)
	})

	t.Run("rejects an unresolvable url", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})

		_, err := svc.AddCompanyBoard(t.Context(), "user-1", dto.AddCompanyBoardInput{CompanyID: company.ID, URL: "https://example.com/careers"})

		assertKind(t, err, apperr.KindUnprocessable)
	})

	t.Run("returns conflict when the board belongs to another company", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		_, _ = companyStore.UpsertCandidateBoard(t.Context(), "company-other", "greenhouse", "acme")
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme2", Name: "Acme2"})

		_, err := svc.AddCompanyBoard(t.Context(), "user-1", dto.AddCompanyBoardInput{CompanyID: company.ID, URL: "https://boards.greenhouse.io/acme"})

		assertKind(t, err, apperr.KindConflict)
	})

	t.Run("adds a board as a candidate without confirming", func(t *testing.T) {
		svc, companyStore := newCompanyService(queuetest.NewRecorder())
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})

		board, err := svc.AddCompanyBoard(t.Context(), "user-1", dto.AddCompanyBoardInput{CompanyID: company.ID, URL: "https://boards.greenhouse.io/acme"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if board.Status != dto.BoardCandidate {
			t.Errorf("status = %v, want candidate", board.Status)
		}
	})

	t.Run("queues verification when confirmed and leaves the board a candidate", func(t *testing.T) {
		q := queuetest.NewRecorder()
		svc, companyStore := newCompanyService(q)
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})

		board, err := svc.AddCompanyBoard(t.Context(), "user-1", dto.AddCompanyBoardInput{CompanyID: company.ID, URL: "https://boards.greenhouse.io/acme", Confirm: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if board.Status != dto.BoardCandidate {
			t.Errorf("status = %v, want candidate", board.Status)
		}
		tasks := q.Tasks()
		if len(tasks) != 1 {
			t.Fatalf("published %d tasks, want 1", len(tasks))
		}
		got := tasks[0]
		if got.Kind != queue.BoardVerifyTask || got.Source != "greenhouse" || got.CompanyID != company.ID || got.BoardToken != "acme" {
			t.Errorf("task = %+v", got)
		}
	})

	t.Run("does not queue verification without confirming", func(t *testing.T) {
		q := queuetest.NewRecorder()
		svc, companyStore := newCompanyService(q)
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})

		if _, err := svc.AddCompanyBoard(t.Context(), "user-1", dto.AddCompanyBoardInput{CompanyID: company.ID, URL: "https://boards.greenhouse.io/acme"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n := len(q.Tasks()); n != 0 {
			t.Errorf("published %d tasks, want 0", n)
		}
	})

	t.Run("returns the publish error when the queue rejects the task", func(t *testing.T) {
		wantErr := errors.New("broker down")
		svc, companyStore := newCompanyService(failingPublisher{err: wantErr})
		company, _ := companyStore.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})

		_, err := svc.AddCompanyBoard(t.Context(), "user-1", dto.AddCompanyBoardInput{CompanyID: company.ID, URL: "https://boards.greenhouse.io/acme", Confirm: true})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want broker error", err)
		}
	})
}

func assertKind(t *testing.T, err error, want apperr.Kind) {
	t.Helper()
	status, ok := apperr.StatusFor(err)
	if !ok {
		t.Fatalf("want kinded error, got %v", err)
	}
	if wantStatus := want.Status(); status != wantStatus {
		t.Errorf("status = %d, want %d", status, wantStatus)
	}
}
