package api_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ollymarsters/job-scraper/internal/api"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

const ingestTestToken = "router-test-ingest-token" //nolint:gosec // test-only static token, not a credential

const nilUUID = "00000000-0000-0000-0000-000000000000"

var (
	router        http.Handler
	testPool      *pgxpool.Pool
	testBroker    = &queue.Broker{}
	testScoring   *scoring.Module
	testApps      *applications.Module
	testIdentity  *identity.Module
	testJobsearch *jobsearch.Module
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot determine test file path")
	}
	moduleRoot := filepath.Join(filepath.Dir(filename), "../..")
	if err := os.Chdir(moduleRoot); err != nil {
		log.Fatalf("chdir to module root: %v", err)
	}

	pgCont, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}

	connStr, err := pgCont.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("connection string: %v", err)
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("pgxpool.New: %v", err)
	}
	if err := testPool.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	if err := data.RunMigrations(ctx, testPool); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	if err := os.Setenv("INGEST_SERVICE_TOKEN", ingestTestToken); err != nil {
		log.Fatalf("set INGEST_SERVICE_TOKEN: %v", err)
	}
	if err := os.Setenv("AI_CREDENTIAL_ENC_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32))); err != nil {
		log.Fatalf("set AI_CREDENTIAL_ENC_KEY: %v", err)
	}

	testApps = applications.New(testPool)
	testIdentity, err = identity.New(testPool, testApps,
		os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("GOOGLE_REDIRECT_URL"))
	if err != nil {
		log.Fatalf("identity.New: %v", err)
	}

	testJobsearch = jobsearch.New(testPool, testBroker, scoring.NewFacade(testPool))
	testScoring = scoring.New(testPool, testIdentity, testIdentity, testJobsearch, "", "")
	testCVTemplates := cvtemplates.New(testPool, testIdentity.DocsClient())
	router = api.NewRouter(testIdentity, testJobsearch, testApps, testCVTemplates, testScoring)

	code := m.Run()

	testPool.Close()
	if err := pgCont.Terminate(ctx); err != nil {
		log.Printf("terminate container: %v", err)
	}
	os.Exit(code)
}

func do(req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return bytes.NewReader(b)
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(w.Body).Decode(&v); err != nil {
		t.Fatalf("decode response %q: %v", w.Body.String(), err)
	}
	return v
}

func signup(t *testing.T) *http.Cookie {
	t.Helper()
	body := jsonBody(t, map[string]string{
		"username": fmt.Sprintf("router-test-%d", time.Now().UnixNano()),
		"password": "correct horse battery staple",
		"email":    "router-test@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", body)
	w := do(req)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup: status = %d, body = %s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_id" {
			return c
		}
	}
	t.Fatal("signup did not set a session cookie")
	return nil
}

func authed(method, path string, body *bytes.Reader, cookie *http.Cookie) *http.Request {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(cookie)
	return req
}

func TestRouterRequiresAuth(t *testing.T) {
	routes := []struct {
		method, path string
	}{
		{http.MethodGet, "/jobs"},
		{http.MethodGet, "/jobs/all"},
		{http.MethodGet, "/jobs/x"},
		{http.MethodGet, "/application-statuses/"},
		{http.MethodPost, "/application-statuses/"},
		{http.MethodPatch, "/application-statuses/x"},
		{http.MethodDelete, "/application-statuses/x"},
		{http.MethodGet, "/applications/"},
		{http.MethodPost, "/applications/"},
		{http.MethodPatch, "/applications/x"},
		{http.MethodDelete, "/applications/x"},
		{http.MethodGet, "/applications/for-jobs"},
		{http.MethodGet, "/sources"},
		{http.MethodGet, "/sources/resolve"},
		{http.MethodGet, "/profile"},
		{http.MethodPut, "/profile"},
		{http.MethodGet, "/scoring-config"},
		{http.MethodPut, "/scoring-config"},
		{http.MethodGet, "/scoring-options"},
		{http.MethodGet, "/scores/status"},
		{http.MethodPost, "/scores/recompute"},
		{http.MethodGet, "/ai-prefs"},
		{http.MethodPut, "/ai-credentials"},
		{http.MethodGet, "/google/oauth/callback"},
		{http.MethodGet, "/google/status"},
		{http.MethodDelete, "/google/link"},
		{http.MethodGet, "/source-targets/"},
		{http.MethodPost, "/source-targets/"},
		{http.MethodPatch, "/source-targets/x"},
		{http.MethodPost, "/source-targets/x/scrape"},
		{http.MethodDelete, "/source-targets/x"},
		{http.MethodGet, "/companies/"},
		{http.MethodPost, "/companies/"},
		{http.MethodPut, "/companies/x/tracking"},
		{http.MethodGet, "/companies/x/boards"},
		{http.MethodPost, "/companies/x/boards"},
		{http.MethodGet, "/cv-templates/"},
		{http.MethodGet, "/cv-templates/x/y/pdf"},
		{http.MethodPost, "/tracked-docs/"},
		{http.MethodDelete, "/tracked-docs/x"},
		{http.MethodPost, "/tracked-docs/x/tabs/y/hide"},
		{http.MethodPost, "/tracked-docs/x/tabs/y/show"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			w := do(httptest.NewRequest(rt.method, rt.path, nil))
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401 (body: %s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestAccessLogWiredIntoRouter(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	do(httptest.NewRequest(http.MethodGet, "/jobs", nil))

	if !strings.Contains(buf.String(), `"event":"`+telemetry.EventHTTPRequest+`"`) {
		t.Fatalf("expected an %s log line, got: %s", telemetry.EventHTTPRequest, buf.String())
	}
}

func TestIngestRequiresServiceToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewReader([]byte(`{}`)))
	w := do(req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("without token: status = %d, want 401", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+ingestTestToken)
	w = do(req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("with token: status = %d, want 400 (body: %s)", w.Code, w.Body.String())
	}
}

func TestRouterRoutes(t *testing.T) {
	cookie := signup(t)

	t.Run("jobs", func(t *testing.T) {
		if w := do(authed(http.MethodGet, "/jobs", nil, cookie)); w.Code != http.StatusOK {
			t.Errorf("GET /jobs = %d", w.Code)
		}
		if w := do(authed(http.MethodGet, "/jobs/all", nil, cookie)); w.Code != http.StatusOK {
			t.Errorf("GET /jobs/all = %d", w.Code)
		}
	})

	t.Run("application-statuses", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/application-statuses/", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET / = %d", w.Code)
		}
		if seeded := decode[[]struct{ ID string }](t, w); len(seeded) != 5 {
			t.Fatalf("default statuses from signup = %d, want 5", len(seeded))
		}
		w = do(authed(http.MethodPost, "/application-statuses/", jsonBody(t, map[string]string{"name": "Offer", "colour": "#00ff00"}), cookie))
		if w.Code != http.StatusCreated {
			t.Fatalf("POST / = %d (body: %s)", w.Code, w.Body.String())
		}
		id := decode[struct{ ID string }](t, w).ID

		w = do(authed(http.MethodPatch, "/application-statuses/"+id, jsonBody(t, map[string]string{"name": "Offer!", "colour": "#00ff00"}), cookie))
		if w.Code != http.StatusOK {
			t.Errorf("PATCH /{id} = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodDelete, "/application-statuses/"+id, nil, cookie))
		if w.Code != http.StatusNoContent {
			t.Errorf("DELETE /{id} = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("applications", func(t *testing.T) {
		var jobID string
		err := testPool.QueryRow(context.Background(),
			`INSERT INTO jobs (title, location, url, company_slug, source, updated_at)
			 VALUES ('Engineer', 'Remote', 'https://example.com/router-test-applications', 'acme', 'greenhouse', NOW())
			 RETURNING id`).Scan(&jobID)
		if err != nil {
			t.Fatalf("insert job: %v", err)
		}

		w := do(authed(http.MethodGet, "/applications/", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET / = %d", w.Code)
		}
		w = do(authed(http.MethodPost, "/applications/", jsonBody(t, map[string]string{"job_id": jobID}), cookie))
		if w.Code != http.StatusCreated {
			t.Fatalf("POST / = %d (body: %s)", w.Code, w.Body.String())
		}
		id := decode[struct{ ID string }](t, w).ID

		w = do(authed(http.MethodPatch, "/applications/"+id, jsonBody(t, map[string]string{"notes": "applied"}), cookie))
		if w.Code != http.StatusOK {
			t.Errorf("PATCH /{id} = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodGet, "/applications/for-jobs?job_ids="+jobID, nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("GET /for-jobs = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodDelete, "/applications/"+id, nil, cookie))
		if w.Code != http.StatusNoContent {
			t.Errorf("DELETE /{id} = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("source-targets", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/source-targets/", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET / = %d", w.Code)
		}
		w = do(authed(http.MethodPost, "/source-targets/", jsonBody(t, map[string]any{
			"source": "greenhouse", "value": "acmecorp", "enabled": false,
		}), cookie))
		if w.Code != http.StatusCreated {
			t.Fatalf("POST / = %d (body: %s)", w.Code, w.Body.String())
		}
		id := decode[struct{ ID string }](t, w).ID

		w = do(authed(http.MethodPatch, "/source-targets/"+id, jsonBody(t, map[string]any{"check_interval_minutes": 90}), cookie))
		if w.Code != http.StatusOK {
			t.Errorf("PATCH /{id} = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodPost, "/source-targets/"+id+"/scrape", nil, cookie))
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("POST /{id}/scrape = %d, want 503 (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodDelete, "/source-targets/"+id, nil, cookie))
		if w.Code != http.StatusNoContent {
			t.Errorf("DELETE /{id} = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	var companyID string
	t.Run("companies", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/companies/", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET / = %d", w.Code)
		}
		w = do(authed(http.MethodPost, "/companies/", jsonBody(t, map[string]any{
			"url": "https://boards.greenhouse.io/acmecorp", "track": false,
		}), cookie))
		if w.Code != http.StatusCreated {
			t.Fatalf("POST / = %d (body: %s)", w.Code, w.Body.String())
		}
		companyID = decode[struct{ ID string }](t, w).ID

		w = do(authed(http.MethodPut, "/companies/"+companyID+"/tracking", jsonBody(t, map[string]any{
			"enabled": true, "check_interval_minutes": 90,
		}), cookie))
		if w.Code != http.StatusOK {
			t.Errorf("PUT /{id}/tracking = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodGet, "/companies/"+companyID+"/boards", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("GET /{id}/boards = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodPost, "/companies/"+companyID+"/boards", jsonBody(t, map[string]any{
			"url": "https://boards.greenhouse.io/otherco", "confirm": false,
		}), cookie))
		if w.Code != http.StatusCreated {
			t.Errorf("POST /{id}/boards = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodGet, "/companies/"+nilUUID+"/boards", nil, cookie))
		if w.Code != http.StatusNotFound {
			t.Errorf("GET /{id}/boards (unknown id) = %d, want 404", w.Code)
		}
	})

	t.Run("profile", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/profile", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("GET = %d", w.Code)
		}
		w = do(authed(http.MethodPut, "/profile", jsonBody(t, map[string]string{"email": "new@example.com"}), cookie))
		if w.Code != http.StatusNoContent {
			t.Errorf("PUT = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("google link", func(t *testing.T) {
		w := do(httptest.NewRequest(http.MethodGet, "/google/oauth/start", nil))
		if w.Code != http.StatusTemporaryRedirect {
			t.Fatalf("GET /google/oauth/start = %d, want 307", w.Code)
		}
		if loc := w.Header().Get("Location"); !strings.Contains(loc, "accounts.google.com") {
			t.Errorf("Location = %q, want a Google auth URL", loc)
		}
		var stateCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == "oauth_state" {
				stateCookie = c
			}
		}
		if stateCookie == nil {
			t.Fatal("expected oauth_state cookie to be set")
		}

		w = do(authed(http.MethodGet, "/google/status", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /google/status = %d (body: %s)", w.Code, w.Body.String())
		}
		status := decode[map[string]any](t, w)
		if status["connected"] != false {
			t.Errorf("connected = %v, want false (no token stored)", status["connected"])
		}

		w = do(authed(http.MethodDelete, "/google/link", nil, cookie))
		if w.Code != http.StatusNoContent {
			t.Errorf("DELETE /google/link = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("scoring-config", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/scoring-config", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("GET = %d", w.Code)
		}
		w = do(authed(http.MethodPut, "/scoring-config", jsonBody(t, map[string]any{
			"notifyThreshold": 5,
			"preferences":     map[string]any{"picks": []any{}},
		}), cookie))
		if w.Code != http.StatusOK {
			t.Errorf("PUT = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodGet, "/scoring-options", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("GET /scoring-options = %d", w.Code)
		}
	})

	t.Run("scores", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/scores/status", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("GET /status = %d", w.Code)
		}
		w = do(authed(http.MethodPost, "/scores/recompute", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("POST /recompute = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("ai-credentials and ai-prefs", func(t *testing.T) {
		key := "sk-test"
		w := do(authed(http.MethodPut, "/ai-credentials", jsonBody(t, map[string]any{"provider": "openrouter", "apiKey": &key}), cookie))
		if w.Code != http.StatusNoContent {
			t.Fatalf("PUT /ai-credentials = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodGet, "/ai-prefs", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /ai-prefs = %d", w.Code)
		}
		w = do(authed(http.MethodPut, "/ai-credentials", jsonBody(t, map[string]any{"provider": "openrouter", "apiKey": nil}), cookie))
		if w.Code != http.StatusNoContent {
			t.Errorf("PUT /ai-credentials (delete) = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("ai-credentials accepts an openrouter key", func(t *testing.T) {
		key := "sk-or-v1-test"
		w := do(authed(http.MethodPut, "/ai-credentials", jsonBody(t, map[string]any{"provider": "openrouter", "apiKey": &key}), cookie))
		if w.Code != http.StatusNoContent {
			t.Fatalf("PUT /ai-credentials = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodGet, "/ai-prefs", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /ai-prefs = %d", w.Code)
		}
		prefs := decode[map[string]any](t, w)
		configured, _ := prefs["configuredProviders"].([]any)
		if !slices.Contains(configured, "openrouter") {
			t.Errorf("configuredProviders = %v, want to contain openrouter", configured)
		}
		w = do(authed(http.MethodPut, "/ai-credentials", jsonBody(t, map[string]any{"provider": "openrouter", "apiKey": nil}), cookie))
		if w.Code != http.StatusNoContent {
			t.Errorf("PUT /ai-credentials (delete) = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("cv-templates and tracked-docs", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/cv-templates/", nil, cookie))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET /cv-templates/ = %d, want 401 (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodGet, "/cv-templates/doc1/tab1/pdf", nil, cookie))
		if w.Code != http.StatusBadGateway {
			t.Errorf("GET /cv-templates/{docId}/{tabId}/pdf = %d, want 502 (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodPost, "/tracked-docs/", jsonBody(t, map[string]string{
			"url": "https://docs.google.com/document/d/abc123/edit",
		}), cookie))
		if w.Code != http.StatusNotFound {
			t.Errorf("POST /tracked-docs/ = %d, want 404 (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodDelete, "/tracked-docs/doc1", nil, cookie))
		if w.Code != http.StatusNotFound {
			t.Errorf("DELETE /tracked-docs/{id} = %d, want 404 (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodPost, "/tracked-docs/doc1/tabs/tab1/hide", nil, cookie))
		if w.Code != http.StatusNotFound {
			t.Errorf("POST /tracked-docs/{docId}/tabs/{tabId}/hide = %d, want 404 (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodPost, "/tracked-docs/doc1/tabs/tab1/show", nil, cookie))
		if w.Code != http.StatusNotFound {
			t.Errorf("POST /tracked-docs/{docId}/tabs/{tabId}/show = %d, want 404 (body: %s)", w.Code, w.Body.String())
		}
	})
}

type fakeModule struct{}

func (fakeModule) Routes(r chi.Router) {
	r.Get("/fake-private", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func (fakeModule) PublicRoutes(r chi.Router) {
	r.Get("/fake-public", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestRouterMountsModules(t *testing.T) {
	moduleRouter := api.NewRouter(testIdentity, testJobsearch, testApps, testScoring, fakeModule{})

	w := httptest.NewRecorder()
	moduleRouter.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/fake-private", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET /fake-private without session = %d, want 401", w.Code)
	}

	cookie := signup(t)
	req := authed(http.MethodGet, "/fake-private", nil, cookie)
	w = httptest.NewRecorder()
	moduleRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /fake-private with session = %d, want 200", w.Code)
	}

	w = httptest.NewRecorder()
	moduleRouter.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/fake-public", nil))
	if w.Code != http.StatusOK {
		t.Errorf("GET /fake-public without session = %d, want 200", w.Code)
	}
}
