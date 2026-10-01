package jobsearch_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

const (
	userID    = "user-1"
	acmeBoard = "https://boards.greenhouse.io/acme"
)

func newCompanyService(q jobsearch.QueuePublisher) (*jobsearch.Service, *jobsearchtest.FakeStore) {
	st := jobsearchtest.NewFakeStore()
	return jobsearch.NewService(st, q), st
}

func seedCompany(t *testing.T, st *jobsearchtest.FakeStore, in dto.CompanyUpsert) dto.Company {
	t.Helper()
	company, err := st.UpsertCompany(t.Context(), in)
	if err != nil {
		t.Fatalf("UpsertCompany(%+v) err = %v", in, err)
	}
	return company
}

func seedAcme(t *testing.T, st *jobsearchtest.FakeStore) dto.Company {
	t.Helper()
	return seedCompany(t, st, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
}

func trackingFor(t *testing.T, st *jobsearchtest.FakeStore, companyID string) (dto.Company, bool) {
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

func TestCreateCompany(t *testing.T) {
	t.Run("resolves the board url", func(t *testing.T) {
		tests := []struct {
			name        string
			in          dto.CreateCompanyInput
			wantTracked bool
		}{
			{name: "tracks a valid board url by default", in: dto.CreateCompanyInput{URL: acmeBoard}, wantTracked: true},
			{name: "does not track when track is false", in: dto.CreateCompanyInput{URL: acmeBoard, Track: new(false)}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, st := newCompanyService(queuetest.NewRecorder())
				company, err := svc.CreateCompany(t.Context(), userID, tt.in)
				if err != nil {
					t.Fatalf("CreateCompany(%+v) err = %v", tt.in, err)
				}
				if company.Slug != "acme" {
					t.Errorf("CreateCompany(%+v).Slug = %q, want acme", tt.in, company.Slug)
				}
				tracked, ok := trackingFor(t, st, company.ID)
				if got := ok && tracked.Tracked; got != tt.wantTracked {
					t.Errorf("tracked = %v, want %v", got, tt.wantTracked)
				}
			})
		}
	})

	t.Run("rejects bad input", func(t *testing.T) {
		tests := []struct {
			name     string
			in       dto.CreateCompanyInput
			wantKind apperr.Kind
		}{
			{name: "missing url", in: dto.CreateCompanyInput{}, wantKind: apperr.KindInvalid},
			{name: "unresolvable url", in: dto.CreateCompanyInput{URL: "https://example.com/careers"}, wantKind: apperr.KindUnprocessable},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, _ := newCompanyService(queuetest.NewRecorder())
				_, err := svc.CreateCompany(t.Context(), userID, tt.in)
				if !apperr.IsKind(err, tt.wantKind) {
					t.Fatalf("CreateCompany(%+v) err = %v, want kind %v", tt.in, err, tt.wantKind)
				}
			})
		}
	})
}

func TestSetCompanyTracking(t *testing.T) {
	t.Run("rejects bad input", func(t *testing.T) {
		tests := []struct {
			name     string
			in       func(companyID string) dto.SetCompanyTrackingInput
			wantKind apperr.Kind
		}{
			{
				name:     "missing enabled field",
				in:       func(id string) dto.SetCompanyTrackingInput { return dto.SetCompanyTrackingInput{CompanyID: id} },
				wantKind: apperr.KindInvalid,
			},
			{
				name: "interval below 60 minutes",
				in: func(id string) dto.SetCompanyTrackingInput {
					return dto.SetCompanyTrackingInput{CompanyID: id, Enabled: new(true), CheckIntervalMinutes: new(30)}
				},
				wantKind: apperr.KindInvalid,
			},
			{
				name: "unknown company",
				in: func(string) dto.SetCompanyTrackingInput {
					return dto.SetCompanyTrackingInput{CompanyID: "missing", Enabled: new(true)}
				},
				wantKind: apperr.KindNotFound,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, st := newCompanyService(queuetest.NewRecorder())
				company := seedAcme(t, st)
				_, err := svc.SetCompanyTracking(t.Context(), userID, tt.in(company.ID))
				if !apperr.IsKind(err, tt.wantKind) {
					t.Fatalf("SetCompanyTracking() err = %v, want kind %v", err, tt.wantKind)
				}
			})
		}
	})

	t.Run("enables tracking for an ATS company without writing a source target", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		company := seedCompany(t, st, dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"})

		tracking, err := svc.SetCompanyTracking(t.Context(), userID, dto.SetCompanyTrackingInput{
			CompanyID: company.ID, Enabled: new(true), CheckIntervalMinutes: new(180),
		})
		if err != nil {
			t.Fatalf("SetCompanyTracking() err = %v", err)
		}
		if !tracking.Enabled || tracking.CheckIntervalMinutes != 180 {
			t.Errorf("tracking = %+v, want enabled every 180 minutes", tracking)
		}
		targets, err := st.ListSourceTargetsByUser(t.Context(), userID)
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 0 {
			t.Errorf("source targets = %+v, want none", targets)
		}
	})

	t.Run("preserves the existing frequency when no interval is given", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		company := seedAcme(t, st)
		enable := dto.SetCompanyTrackingInput{CompanyID: company.ID, Enabled: new(true), CheckIntervalMinutes: new(180)}
		if _, err := svc.SetCompanyTracking(t.Context(), userID, enable); err != nil {
			t.Fatal(err)
		}

		tracking, err := svc.SetCompanyTracking(t.Context(), userID, dto.SetCompanyTrackingInput{CompanyID: company.ID, Enabled: new(false)})
		if err != nil {
			t.Fatalf("SetCompanyTracking() err = %v", err)
		}
		if tracking.Enabled || tracking.CheckIntervalMinutes != 180 {
			t.Errorf("tracking = %+v, want paused at 180 minutes", tracking)
		}
	})
}

func TestListCompanyBoards(t *testing.T) {
	t.Run("returns not found for an unknown company", func(t *testing.T) {
		svc, _ := newCompanyService(queuetest.NewRecorder())
		_, err := svc.ListCompanyBoards(t.Context(), userID, "missing")
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("err = %v, want kind %v", err, apperr.KindNotFound)
		}
	})

	t.Run("lists boards linked to the company", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		company := seedAcme(t, st)
		if _, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "acme"); err != nil {
			t.Fatal(err)
		}

		boards, err := svc.ListCompanyBoards(t.Context(), userID, company.ID)
		if err != nil {
			t.Fatalf("ListCompanyBoards() err = %v", err)
		}
		if len(boards) != 1 {
			t.Errorf("ListCompanyBoards() = %d boards, want 1", len(boards))
		}
	})
}

func TestAddCompanyBoard(t *testing.T) {
	t.Run("rejects bad input", func(t *testing.T) {
		tests := []struct {
			name     string
			url      string
			company  func(t *testing.T, st *jobsearchtest.FakeStore) string
			wantKind apperr.Kind
		}{
			{
				name:     "unknown company",
				url:      acmeBoard,
				company:  func(*testing.T, *jobsearchtest.FakeStore) string { return "missing" },
				wantKind: apperr.KindNotFound,
			},
			{
				name: "unresolvable url",
				url:  "https://example.com/careers",
				company: func(t *testing.T, st *jobsearchtest.FakeStore) string {
					t.Helper()
					return seedAcme(t, st).ID
				},
				wantKind: apperr.KindUnprocessable,
			},
			{
				name: "board belongs to another company",
				url:  acmeBoard,
				company: func(t *testing.T, st *jobsearchtest.FakeStore) string {
					t.Helper()
					if _, err := st.UpsertCandidateBoard(t.Context(), "company-other", "greenhouse", "acme"); err != nil {
						t.Fatal(err)
					}
					return seedAcme(t, st).ID
				},
				wantKind: apperr.KindConflict,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, st := newCompanyService(queuetest.NewRecorder())
				in := dto.AddCompanyBoardInput{CompanyID: tt.company(t, st), URL: tt.url}
				if _, err := svc.AddCompanyBoard(t.Context(), userID, in); !apperr.IsKind(err, tt.wantKind) {
					t.Fatalf("AddCompanyBoard() err = %v, want kind %v", err, tt.wantKind)
				}
			})
		}
	})

	t.Run("adds a candidate board and queues verification only when confirmed", func(t *testing.T) {
		tests := []struct {
			name      string
			confirm   bool
			wantTasks int
		}{
			{name: "without confirming", wantTasks: 0},
			{name: "confirmed", confirm: true, wantTasks: 1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				q := queuetest.NewRecorder()
				svc, st := newCompanyService(q)
				company := seedAcme(t, st)

				board, err := svc.AddCompanyBoard(t.Context(), userID, dto.AddCompanyBoardInput{CompanyID: company.ID, URL: acmeBoard, Confirm: tt.confirm})
				if err != nil {
					t.Fatalf("AddCompanyBoard() err = %v", err)
				}
				if board.Status != dto.BoardCandidate {
					t.Errorf("status = %v, want candidate", board.Status)
				}
				tasks := q.Tasks()
				if len(tasks) != tt.wantTasks {
					t.Fatalf("published %d tasks, want %d", len(tasks), tt.wantTasks)
				}
				if tt.wantTasks == 0 {
					return
				}
				got := tasks[0]
				want := queue.Task{Kind: queue.BoardVerifyTask, Source: "greenhouse", CompanyID: company.ID, BoardToken: "acme"}
				if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(queue.Task{}, "Version", "ID", "BoardID")); diff != "" {
					t.Errorf("published task (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("returns the publish error when the queue rejects the task", func(t *testing.T) {
		wantErr := errors.New("broker down")
		svc, st := newCompanyService(queuetest.PublishFails(wantErr))
		company := seedAcme(t, st)

		_, err := svc.AddCompanyBoard(t.Context(), userID, dto.AddCompanyBoardInput{CompanyID: company.ID, URL: acmeBoard, Confirm: true})
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v, want %v", err, wantErr)
		}
	})
}

func TestListTrackedCompanies(t *testing.T) {
	svc, _ := newCompanyService(queuetest.NewRecorder())
	company, err := svc.CreateCompany(t.Context(), userID, dto.CreateCompanyInput{URL: acmeBoard})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.ListTrackedCompanies(t.Context(), userID)
	if err != nil {
		t.Fatalf("ListTrackedCompanies() err = %v", err)
	}
	if len(got) != 1 || len(got[0].Boards) != 1 {
		t.Fatalf("ListTrackedCompanies() = %+v, want one company with one board", got)
	}
	if got[0].ID != company.ID || got[0].Boards[0].URL != acmeBoard {
		t.Errorf("ListTrackedCompanies() = %+v, want the board URL filled", got[0])
	}
}

func TestUntrackCompany(t *testing.T) {
	svc, _ := newCompanyService(queuetest.NewRecorder())
	err := svc.UntrackCompany(t.Context(), userID, "missing")
	if !apperr.IsKind(err, apperr.KindNotFound) {
		t.Fatalf("UntrackCompany() err = %v, want kind %v", err, apperr.KindNotFound)
	}
}

func TestSetCompanyReview(t *testing.T) {
	t.Run("rejects an unknown state", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		company := seedAcme(t, st)
		_, err := svc.SetCompanyReview(t.Context(), userID, dto.SetCompanyReviewInput{CompanyID: company.ID, State: "bogus"})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Fatalf("SetCompanyReview() err = %v, want kind invalid", err)
		}
	})

	t.Run("untracked company is not found", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		company := seedAcme(t, st)
		_, err := svc.SetCompanyReview(t.Context(), userID, dto.SetCompanyReviewInput{CompanyID: company.ID, State: "kept"})
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("SetCompanyReview() err = %v, want ErrNotFound", err)
		}
	})

	t.Run("dismiss keeps the row disabled and undo re-enables it as new", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		company := seedAcme(t, st)
		if _, err := st.SetCompanyTracking(t.Context(), userID, company.ID, true, 0); err != nil {
			t.Fatal(err)
		}
		for _, tt := range []struct {
			state       string
			wantEnabled bool
		}{{"dismissed", false}, {"new", true}, {"kept", true}} {
			got, err := svc.SetCompanyReview(t.Context(), userID, dto.SetCompanyReviewInput{CompanyID: company.ID, State: tt.state})
			if err != nil {
				t.Fatalf("SetCompanyReview(%s) err = %v", tt.state, err)
			}
			if got.ReviewState != tt.state || got.Enabled != tt.wantEnabled {
				t.Errorf("SetCompanyReview(%s) = %+v, want enabled %v", tt.state, got, tt.wantEnabled)
			}
		}
	})
}
