package jobsearch_test

import (
	"context"
	"errors"
	"strings"
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
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

const (
	userID    = "user-1"
	acmeBoard = "https://boards.greenhouse.io/acme"
)

func newCompanyService(q jobsearch.QueuePublisher) (*jobsearch.Service, *jobsearchtest.FakeStore) {
	st := jobsearchtest.NewFakeStore()
	return jobsearch.NewService(st, q, jobsearchtest.NewNoopScoring()), st
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
	page, err := st.PageCompaniesForUser(t.Context(), userID, dto.CompanyPageOptions{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range page.Items {
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

type invalidIDStore struct{ jobsearch.Store }

func (invalidIDStore) GetCompanyForUser(context.Context, string, string) (dto.Company, error) {
	return dto.Company{}, store.ErrInvalidID
}

func TestGetCompany(t *testing.T) {
	t.Run("returns the company with this user's tracking", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		company := seedAcme(t, st)
		if _, err := st.SetCompanyTracking(t.Context(), userID, company.ID, true, 180); err != nil {
			t.Fatal(err)
		}

		got, err := svc.GetCompany(t.Context(), userID, company.ID)
		if err != nil || got.Name != "Acme" || !got.Tracked {
			t.Fatalf("GetCompany() = %+v, %v, want tracked Acme", got, err)
		}
	})

	t.Run("maps store errors to kinds", func(t *testing.T) {
		tests := []struct {
			name     string
			svc      *jobsearch.Service
			wantKind apperr.Kind
		}{
			{"unknown company", jobsearch.NewService(jobsearchtest.NewFakeStore(), queuetest.NewRecorder(), jobsearchtest.NewNoopScoring()), apperr.KindNotFound},
			{"malformed id", jobsearch.NewService(invalidIDStore{}, queuetest.NewRecorder(), jobsearchtest.NewNoopScoring()), apperr.KindInvalid},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := tt.svc.GetCompany(t.Context(), userID, "x")
				if !apperr.IsKind(err, tt.wantKind) {
					t.Fatalf("GetCompany() err = %v, want kind %v", err, tt.wantKind)
				}
			})
		}
	})
}

func TestListCompanies(t *testing.T) {
	t.Run("pages by offset and reports the total", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		for _, name := range []string{"Alpha", "Beta", "Gamma"} {
			seedCompany(t, st, dto.CompanyUpsert{Slug: strings.ToLower(name), Name: name})
		}

		first, err := svc.ListCompanies(t.Context(), userID, dto.CompaniesQuery{Limit: "2"})
		if err != nil || len(first.Items) != 2 || first.Total != 3 {
			t.Fatalf("ListCompanies(limit 2) = %+v, %v, want two items of total 3", first, err)
		}
		second, err := svc.ListCompanies(t.Context(), userID, dto.CompaniesQuery{Limit: "2", Offset: "2"})
		if err != nil || len(second.Items) != 1 || second.Items[0].Name != "Gamma" || second.Total != 3 {
			t.Fatalf("ListCompanies(offset 2) = %+v, %v, want Gamma of total 3", second, err)
		}
	})

	t.Run("defaults to relevance and honours the alphabetical sort", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		alpha := seedCompany(t, st, dto.CompanyUpsert{Slug: "alpha", Name: "Alpha"})
		beta := seedCompany(t, st, dto.CompanyUpsert{Slug: "beta", Name: "Beta"})
		if _, err := st.SetCompanyTracking(t.Context(), userID, beta.ID, true, 180); err != nil {
			t.Fatal(err)
		}

		relevant, err := svc.ListCompanies(t.Context(), userID, dto.CompaniesQuery{})
		if err != nil || relevant.Items[0].ID != beta.ID {
			t.Fatalf("ListCompanies() = %+v, %v, want tracked Beta first", relevant, err)
		}
		alphabetical, err := svc.ListCompanies(t.Context(), userID, dto.CompaniesQuery{Sort: "alphabetical"})
		if err != nil || alphabetical.Items[0].ID != alpha.ID {
			t.Fatalf("ListCompanies(alphabetical) = %+v, %v, want Alpha first", alphabetical, err)
		}
	})

	t.Run("no-board and favourite filters combine", func(t *testing.T) {
		svc, st := newCompanyService(queuetest.NewRecorder())
		fav := seedCompany(t, st, dto.CompanyUpsert{Slug: "fav", Name: "Fav"})
		favWithBoard := seedCompany(t, st, dto.CompanyUpsert{Slug: "fav-board", Name: "Fav Board"})
		seedCompany(t, st, dto.CompanyUpsert{Slug: "other", Name: "Other"})
		for _, id := range []string{fav.ID, favWithBoard.ID} {
			if err := st.SetCompanyFavourite(t.Context(), userID, id, true); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := st.UpsertCandidateBoard(t.Context(), favWithBoard.ID, "greenhouse", "fav-board"); err != nil {
			t.Fatal(err)
		}

		page, err := svc.ListCompanies(t.Context(), userID, dto.CompaniesQuery{NoBoard: "1", Favourite: "1"})
		if err != nil || len(page.Items) != 1 || page.Items[0].ID != fav.ID || page.Total != 1 {
			t.Fatalf("ListCompanies(no board, favourite) = %+v, %v, want only Fav", page, err)
		}
	})

	t.Run("an empty result is an empty list", func(t *testing.T) {
		svc, _ := newCompanyService(queuetest.NewRecorder())
		page, err := svc.ListCompanies(t.Context(), userID, dto.CompaniesQuery{Q: "nothing"})
		if err != nil || page.Items == nil || len(page.Items) != 0 {
			t.Fatalf("ListCompanies() = %+v, %v, want empty non-nil items", page, err)
		}
	})

	t.Run("rejects bad input", func(t *testing.T) {
		tests := []struct {
			name  string
			query dto.CompaniesQuery
		}{
			{"limit above the maximum", dto.CompaniesQuery{Limit: "101"}},
			{"non-numeric limit", dto.CompaniesQuery{Limit: "x"}},
			{"negative offset", dto.CompaniesQuery{Offset: "-1"}},
			{"unknown sort", dto.CompaniesQuery{Sort: "newest"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, _ := newCompanyService(queuetest.NewRecorder())
				if _, err := svc.ListCompanies(t.Context(), userID, tt.query); !apperr.IsKind(err, apperr.KindInvalid) {
					t.Fatalf("ListCompanies(%+v) err = %v, want invalid", tt.query, err)
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

func TestExcludeCompany(t *testing.T) {
	newModule := func(t *testing.T, scoring *jobsearchtest.NoopScoring) (*jobsearch.Module, *jobsearchtest.FakeStore, dto.Company) {
		t.Helper()
		st := jobsearchtest.NewFakeStore()
		deps := jobsearchtest.NewDeps(st)
		deps.Scoring = scoring
		company := seedAcme(t, st)
		if _, err := st.SetCompanyTracking(t.Context(), userID, company.ID, true, 0); err != nil {
			t.Fatal(err)
		}
		return jobsearch.Build(deps), st, company
	}
	excluded := func(t *testing.T, scoring *jobsearchtest.NoopScoring) []string {
		t.Helper()
		cfg, err := scoring.SearchConfig(t.Context(), userID)
		if err != nil {
			t.Fatal(err)
		}
		return cfg.ExcludedCompanies
	}
	reviewState := func(t *testing.T, st *jobsearchtest.FakeStore) string {
		t.Helper()
		tracked, err := st.ListTrackedCompaniesForUser(t.Context(), userID)
		if err != nil || len(tracked) != 1 {
			t.Fatalf("ListTrackedCompaniesForUser() = %v, %v, want one company", tracked, err)
		}
		return tracked[0].ReviewState
	}

	t.Run("dismisses the company and excludes its name once, however often it repeats", func(t *testing.T) {
		scoring := jobsearchtest.NewNoopScoring()
		m, st, company := newModule(t, scoring)
		in := dto.ExcludeCompanyInput{CompanyID: company.ID}

		first, err := m.ExcludeCompany(t.Context(), userID, in)
		if err != nil || !first.Added {
			t.Fatalf("ExcludeCompany() = %+v, %v, want Added", first, err)
		}
		again, err := m.ExcludeCompany(t.Context(), userID, in)
		if err != nil || again.Added {
			t.Fatalf("ExcludeCompany() again = %+v, %v, want not Added", again, err)
		}

		if got := reviewState(t, st); got != "dismissed" {
			t.Errorf("review state = %q, want dismissed", got)
		}
		if diff := cmp.Diff([]string{"acme"}, excluded(t, scoring)); diff != "" {
			t.Errorf("ExcludedCompanies (-want +got):\n%s", diff)
		}
	})

	t.Run("undo restores new and removes only a name the action added", func(t *testing.T) {
		scoring := jobsearchtest.NewNoopScoring()
		m, st, company := newModule(t, scoring)
		added, err := m.ExcludeCompany(t.Context(), userID, dto.ExcludeCompanyInput{CompanyID: company.ID})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := m.UnexcludeCompany(t.Context(), userID, dto.UnexcludeCompanyInput{CompanyID: company.ID, RemoveName: added.Added}); err != nil {
			t.Fatalf("UnexcludeCompany() err = %v", err)
		}
		if got := reviewState(t, st); got != "new" {
			t.Errorf("review state = %q, want new", got)
		}
		if got := excluded(t, scoring); len(got) != 0 {
			t.Errorf("ExcludedCompanies = %v, want empty", got)
		}
	})

	t.Run("undo keeps a name the user already had", func(t *testing.T) {
		scoring := jobsearchtest.NewNoopScoring()
		scoring.SeedSearchConfig(dto.SearchConfig{UserID: userID, ExcludedCompanies: []string{"acme"}})
		m, _, company := newModule(t, scoring)
		excluded0, err := m.ExcludeCompany(t.Context(), userID, dto.ExcludeCompanyInput{CompanyID: company.ID})
		if err != nil || excluded0.Added {
			t.Fatalf("ExcludeCompany() = %+v, %v, want not Added", excluded0, err)
		}

		if _, err := m.UnexcludeCompany(t.Context(), userID, dto.UnexcludeCompanyInput{CompanyID: company.ID, RemoveName: excluded0.Added}); err != nil {
			t.Fatalf("UnexcludeCompany() err = %v", err)
		}
		if diff := cmp.Diff([]string{"acme"}, excluded(t, scoring)); diff != "" {
			t.Errorf("ExcludedCompanies (-want +got):\n%s", diff)
		}
	})

	t.Run("unknown company is not found", func(t *testing.T) {
		m, _, _ := newModule(t, jobsearchtest.NewNoopScoring())
		_, err := m.ExcludeCompany(t.Context(), userID, dto.ExcludeCompanyInput{CompanyID: "missing"})
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("ExcludeCompany() err = %v, want ErrNotFound", err)
		}
	})
}
