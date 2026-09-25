package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/postgres"

	app "github.com/ollymarsters/job-scraper/internal"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

// This is the ADR 0020 router-level table test: it checks that every route
// is wired to the right shape (adapter kind, path parameters) and status,
// not that every service's business rules are correct (those are unit
// tested against providers.Mock* in each feature package).

const ingestTestToken = "router-test-ingest-token" //nolint:gosec // test-only static token, not a credential

// nilUUID is a validly-formatted but non-existent UUID, for exercising a
// not-found path without hitting an earlier "malformed id" branch.
const nilUUID = "00000000-0000-0000-0000-000000000000"

var router http.Handler

func TestMain(m *testing.M) {
	ctx := context.Background()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot determine test file path")
	}
	moduleRoot := filepath.Join(filepath.Dir(filename), "..")
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

	testDB, err := db.New(ctx, connStr)
	if err != nil {
		log.Fatalf("db.New: %v", err)
	}
	if err := data.RunMigrations(ctx, testDB.Pool()); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	if err := os.Setenv("INGEST_SERVICE_TOKEN", ingestTestToken); err != nil {
		log.Fatalf("set INGEST_SERVICE_TOKEN: %v", err)
	}
	// Zero-value broker: never dialed unless a route actually calls Publish,
	// and every route exercised below avoids that path (see comments).
	router = app.NewRouter(testDB, &queue.Broker{}, newFakeCredStore())

	code := m.Run()

	testDB.Close()
	if err := pgCont.Terminate(ctx); err != nil {
		log.Printf("terminate container: %v", err)
	}
	os.Exit(code)
}

// fakeCredStore is a minimal in-memory credstore.CredentialStore, so
// ai-prefs/ai-credentials routes can be exercised without real encryption.
type fakeCredStore struct {
	mu sync.Mutex
	m  map[[2]string]string
}

func newFakeCredStore() *fakeCredStore { return &fakeCredStore{m: map[[2]string]string{}} }

func (f *fakeCredStore) Save(_ context.Context, userID, provider, plainKey string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[[2]string{userID, provider}] = plainKey
	return nil
}

func (f *fakeCredStore) Get(_ context.Context, userID, provider string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key, ok := f.m[[2]string{userID, provider}]
	if !ok {
		return "", credstore.ErrNotFound
	}
	return key, nil
}

func (f *fakeCredStore) Delete(_ context.Context, userID, provider string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.m, [2]string{userID, provider})
	return nil
}

func (f *fakeCredStore) ListProviders(_ context.Context, userID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for k := range f.m {
		if k[0] == userID {
			out = append(out, k[1])
		}
	}
	return out, nil
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

// signup creates a fresh user via the real /auth/signup route and returns
// their session cookie, so authenticated route tests exercise the real
// auth.Middleware wiring rather than a synthetic session.
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

// TestRouterRequiresAuth checks that every route inside the session-auth
// group actually sits behind auth.Middleware: hit without a session cookie,
// each must 401 rather than reach its handler. This is the cheapest and
// highest-value regression guard against a route being wired outside its
// intended group.
func TestRouterRequiresAuth(t *testing.T) {
	routes := []struct {
		method, path string
	}{
		{http.MethodGet, "/jobs"},
		{http.MethodGet, "/jobs/all"},
		{http.MethodGet, "/jobs/x"},
		{http.MethodPost, "/jobs/x/reasoning"},
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
		{http.MethodGet, "/scores/status"},
		{http.MethodPost, "/scores/rescore"},
		{http.MethodGet, "/ai-prefs"},
		{http.MethodPut, "/ai-prefs"},
		{http.MethodPut, "/ai-credentials"},
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

func TestIngestRequiresServiceToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewReader([]byte(`{}`)))
	w := do(req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("without token: status = %d, want 401", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/ingest/batch", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+ingestTestToken)
	w = do(req)
	// The token gate passed; an empty batch is then rejected by the handler.
	if w.Code != http.StatusBadRequest {
		t.Fatalf("with token: status = %d, want 400 (body: %s)", w.Code, w.Body.String())
	}
}

// TestRouterRoutes exercises each authenticated route with a minimal valid
// (or deliberately absent) input and checks the status and path-parameter
// wiring the ADR calls for. It shares one signed-up user across subtests,
// chaining IDs where a route needs one that only another route can create.
func TestRouterRoutes(t *testing.T) {
	cookie := signup(t)

	t.Run("jobs", func(t *testing.T) {
		if w := do(authed(http.MethodGet, "/jobs", nil, cookie)); w.Code != http.StatusOK {
			t.Errorf("GET /jobs = %d", w.Code)
		}
		if w := do(authed(http.MethodGet, "/jobs/all", nil, cookie)); w.Code != http.StatusOK {
			t.Errorf("GET /jobs/all = %d", w.Code)
		}
		// No such job scored yet: exercises the ID adapter's path param and
		// the service's first validation branch, no fixture data needed.
		w := do(authed(http.MethodPost, "/jobs/"+nilUUID+"/reasoning", nil, cookie))
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("POST /jobs/{id}/reasoning = %d, want 422 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("application-statuses", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/application-statuses/", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET / = %d", w.Code)
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

	t.Run("source-targets", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/source-targets/", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET / = %d", w.Code)
		}
		// enabled:false so Create never reaches board verification or the queue.
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
		// No verified board exists, so this fails past the ID lookup —
		// proving the path param and service wiring without a live queue.
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

	t.Run("scoring-config", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/scoring-config", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("GET = %d", w.Code)
		}
		w = do(authed(http.MethodPut, "/scoring-config", jsonBody(t, map[string]any{
			"suitabilityRubric": "test rubric", "notifyThreshold": 5,
		}), cookie))
		if w.Code != http.StatusOK {
			t.Errorf("PUT = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("scores", func(t *testing.T) {
		w := do(authed(http.MethodGet, "/scores/status", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("GET /status = %d", w.Code)
		}
		w = do(authed(http.MethodPost, "/scores/rescore", nil, cookie))
		if w.Code != http.StatusOK {
			t.Errorf("POST /rescore = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("ai-credentials and ai-prefs", func(t *testing.T) {
		key := "sk-test"
		w := do(authed(http.MethodPut, "/ai-credentials", jsonBody(t, map[string]any{"provider": "anthropic", "apiKey": &key}), cookie))
		if w.Code != http.StatusNoContent {
			t.Fatalf("PUT /ai-credentials = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodGet, "/ai-prefs", nil, cookie))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /ai-prefs = %d", w.Code)
		}
		w = do(authed(http.MethodPut, "/ai-prefs", jsonBody(t, map[string]string{
			"suitabilityModel": "claude-haiku-4-5-20251001", "reasoningModel": "claude-sonnet-4-6",
		}), cookie))
		if w.Code != http.StatusOK {
			t.Errorf("PUT /ai-prefs = %d (body: %s)", w.Code, w.Body.String())
		}
		w = do(authed(http.MethodPut, "/ai-credentials", jsonBody(t, map[string]any{"provider": "anthropic", "apiKey": nil}), cookie))
		if w.Code != http.StatusNoContent {
			t.Errorf("PUT /ai-credentials (delete) = %d (body: %s)", w.Code, w.Body.String())
		}
	})

	// No Google account is linked for this user, so every cv-templates /
	// tracked-docs route below fails deterministically past the path-param
	// and body-decode stage — exactly what this table test needs to check,
	// without a real Google API call.
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
		// docId/tabId are two distinct path params (the ID2 adapter) — this
		// is the case a same-named {id} param would silently break.
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
