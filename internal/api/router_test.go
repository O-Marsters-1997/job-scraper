package api_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/api"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

const ingestToken = "router-test-ingest-token"

type app struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router http.Handler
}

func newApp(t *testing.T) *app {
	t.Helper()
	t.Setenv("INGEST_SERVICE_TOKEN", ingestToken)
	t.Setenv("AI_CREDENTIAL_ENC_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("GOOGLE_TOKEN_ENC_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))

	pool := pgtest.New(t)
	apps := applications.New(pool)
	idm, err := identity.New(pool, apps, "", "", "")
	if err != nil {
		t.Fatalf("identity.New: %v", err)
	}
	js := jobsearch.New(pool, &queue.Broker{}, scoring.NewFacade(pool))
	sc := scoring.New(pool, idm, idm, js, "", "")
	cv := cvtemplates.New(pool, idm.DocsClient())
	return &app{t: t, pool: pool, router: api.NewRouter(idm, js, apps, cv, sc)}
}

type user struct {
	*app
	cookie *http.Cookie
}

func (a *app) signup() user {
	a.t.Helper()
	req := jsonRequest(a.t, http.MethodPost, "/auth/signup", map[string]string{
		"username": "journey-user",
		"password": "correct horse battery staple",
		"email":    "journey@example.com",
	})
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		a.t.Fatalf("signup = %d: %s", w.Code, w.Body)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_id" {
			return user{app: a, cookie: c}
		}
	}
	a.t.Fatal("signup set no session cookie")
	return user{}
}

func jsonRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var r io.Reader = http.NoBody
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func send[T any](a *app, wantStatus int, req *http.Request) T {
	a.t.Helper()
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, req)
	if w.Code != wantStatus {
		a.t.Fatalf("%s %s = %d, want %d: %s", req.Method, req.URL.Path, w.Code, wantStatus, w.Body)
	}
	var v T
	if w.Body.Len() > 0 {
		v = handlerstest.DecodeJSON[T](a.t, w.Body.Bytes())
	}
	return v
}

func do[T any](u user, wantStatus int, method, path string, body any) T {
	u.t.Helper()
	req := jsonRequest(u.t, method, path, body)
	req.AddCookie(u.cookie)
	return send[T](u.app, wantStatus, req)
}

type ingestResult struct {
	Status string `json:"status"`
	JobID  string `json:"job_id"`
}

func (a *app) ingest(job map[string]string) ingestResult {
	a.t.Helper()
	req := jsonRequest(a.t, http.MethodPost, "/ingest", job)
	req.Header.Set("Authorization", "Bearer "+ingestToken)
	return send[ingestResult](a, http.StatusOK, req)
}

func TestSignupSeedsDefaultStatuses(t *testing.T) {
	u := newApp(t).signup()

	statuses := do[[]dto.ApplicationStatus](u, http.StatusOK, http.MethodGet, "/application-statuses/", nil)

	if len(statuses) != 5 {
		t.Fatalf("default statuses = %d, want 5", len(statuses))
	}
}

func TestIngestedJobCanBeAppliedToAndMovedThroughStatuses(t *testing.T) {
	a := newApp(t)
	u := a.signup()

	res := a.ingest(map[string]string{
		"title": "Engineer", "location": "Remote", "url": "https://example.com/jobs/1",
		"company_slug": "acme", "source": "greenhouse",
	})
	if res.JobID == "" {
		t.Fatalf("ingest returned no job id (status %q)", res.Status)
	}

	page := do[dto.JobPage](u, http.StatusOK, http.MethodGet, "/jobs", nil)
	if len(page.Items) != 1 || page.Items[0].ID != res.JobID {
		t.Fatalf("GET /jobs items = %+v, want the ingested job %s", page.Items, res.JobID)
	}

	created := do[dto.Application](u, http.StatusCreated, http.MethodPost, "/applications/", map[string]string{"job_id": res.JobID})
	statuses := do[[]dto.ApplicationStatus](u, http.StatusOK, http.MethodGet, "/application-statuses/", nil)
	var next string
	for _, s := range statuses {
		if s.ID != created.StatusID {
			next = s.ID
			break
		}
	}
	do[dto.Application](u, http.StatusOK, http.MethodPatch, "/applications/"+created.ID, map[string]string{"status_id": next})

	got := do[[]dto.ApplicationWithDetails](u, http.StatusOK, http.MethodGet, "/applications/", nil)
	if len(got) != 1 || got[0].StatusID != next {
		t.Fatalf("applications = %+v, want one in status %s", got, next)
	}
}

func TestRecomputeAfterConfigUpdateShowsScoresOnJobs(t *testing.T) {
	a := newApp(t)
	u := a.signup()

	job := a.ingest(map[string]string{
		"title": "Engineer", "location": "Remote", "url": "https://example.com/jobs/2",
		"company_slug": "acme", "source": "greenhouse", "salaryraw": "£30,000",
	})
	if _, err := a.pool.Exec(t.Context(),
		"INSERT INTO job_scores (job_id, user_id) SELECT $1, id FROM users", job.JobID); err != nil {
		t.Fatalf("seed job score row: %v", err)
	}
	if page := do[dto.JobPage](u, http.StatusOK, http.MethodGet, "/jobs", nil); page.Items[0].SuitabilityScore != nil {
		t.Fatalf("score before recompute = %d, want none", *page.Items[0].SuitabilityScore)
	}

	do[struct{}](u, http.StatusOK, http.MethodPut, "/scoring-config", map[string]any{
		"notifyThreshold": 50,
		"preferences":     map[string]any{"picks": []any{}, "salaryFloor": map[string]any{"amount": 60000, "currency": "GBP"}},
	})
	recomputed := do[dto.RecomputeResult](u, http.StatusOK, http.MethodPost, "/scores/recompute", nil)
	if recomputed.Recomputed != 1 {
		t.Fatalf("recomputed = %d, want 1", recomputed.Recomputed)
	}

	page := do[dto.JobPage](u, http.StatusOK, http.MethodGet, "/jobs", nil)
	if page.Items[0].SuitabilityScore == nil {
		t.Fatal("job has no score after recompute")
	}
}

func TestTrackedCompanyGetsBoard(t *testing.T) {
	u := newApp(t).signup()

	company := do[dto.Company](u, http.StatusCreated, http.MethodPost, "/companies/", map[string]any{
		"url": "https://boards.greenhouse.io/acmecorp", "track": true,
	})
	do[struct{}](u, http.StatusOK, http.MethodPut, "/companies/"+company.ID+"/tracking", map[string]any{"enabled": true})

	boards := do[[]dto.CompanyBoard](u, http.StatusOK, http.MethodGet, "/companies/"+company.ID+"/boards", nil)
	if len(boards) != 1 || boards[0].Source != "greenhouse" || boards[0].BoardToken != "acmecorp" {
		t.Fatalf("boards = %+v, want the greenhouse/acmecorp board", boards)
	}
}
