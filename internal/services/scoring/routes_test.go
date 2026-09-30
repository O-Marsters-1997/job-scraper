package scoring_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
)

func newTestRouter(t *testing.T, st *scoringtest.FakeStore) chi.Router {
	t.Helper()
	m := scoring.Build(newDeps(t, st))
	r := chi.NewRouter()
	m.Routes(r)
	return r
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newTestRouter(t, newFakeStore())

	handlerstest.RequiresAuth(t, r,
		"GET /scoring-config",
		"PUT /scoring-config",
		"GET /scoring-options",
		"GET /scores/status",
		"POST /scores/recompute",
	)
	handlerstest.RejectsMalformedBody(t, r, "PUT /scoring-config")
}

func TestScoringConfigRoute(t *testing.T) {
	st := newFakeStore()
	st.SeedSearchConfig(dto.SearchConfig{UserID: handlerstest.UserID, NotifyThreshold: 70})
	r := newTestRouter(t, st)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, handlerstest.Request(t, http.MethodGet, "/scoring-config", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), `"notifyThreshold":70`) {
		t.Fatalf("body = %s, want notifyThreshold 70", w.Body)
	}
}

func TestScoringConfigUpdateRoute(t *testing.T) {
	r := newTestRouter(t, newFakeStore())

	body := `{"notifyThreshold":80,"excludedTitleKeywords":[],"excludedCompanies":[],"excludedLocations":[],
		"preferences":{"picks":[{"optionId":"tech:go","stance":"nice"}]}}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, handlerstest.Request(t, http.MethodPut, "/scoring-config", body))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), `"notifyThreshold":80`) {
		t.Fatalf("body = %s, want notifyThreshold 80", w.Body)
	}
	if !strings.Contains(w.Body.String(), `"optionId":"tech:go"`) {
		t.Fatalf("body = %s, want the saved pick", w.Body)
	}
}

func TestScoringOptionsRoute(t *testing.T) {
	r := newTestRouter(t, newFakeStore())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, handlerstest.Request(t, http.MethodGet, "/scoring-options", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), `"id":"tech:go"`) {
		t.Fatalf("body = %s, want the seeded tech:go option", w.Body)
	}
}

func TestScoresStatusRoute(t *testing.T) {
	r := newTestRouter(t, newFakeStore())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, handlerstest.Request(t, http.MethodGet, "/scores/status", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), `"pending":0`) {
		t.Fatalf("body = %s, want pending 0", w.Body)
	}
}

func TestScoresRecomputeRoute(t *testing.T) {
	st := newFakeStore()
	st.SeedSearchConfig(dto.SearchConfig{UserID: handlerstest.UserID})
	r := newTestRouter(t, st)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, handlerstest.Request(t, http.MethodPost, "/scores/recompute", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), `"recomputed":0`) {
		t.Fatalf("body = %s, want recomputed 0 (no scored jobs)", w.Body)
	}
}
