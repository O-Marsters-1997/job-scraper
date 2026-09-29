package tailoring_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/tailoring"
	"github.com/ollymarsters/job-scraper/internal/services/tailoring/tailoringtest"
)

func newTestRouter() chi.Router {
	m := tailoring.Build(tailoring.Deps{Store: tailoringtest.NewFakeStore()})
	r := chi.NewRouter()
	m.Routes(r)
	return r
}

func do(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := handlerstest.Authed(httptest.NewRequest(method, path, strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newTestRouter()
	handlerstest.RequiresAuth(t, r,
		"GET /experience/",
		"POST /experience/positions",
		"PUT /experience/positions/order",
		"PATCH /experience/positions/{id}",
		"DELETE /experience/positions/{id}",
		"POST /experience/positions/{id}/achievements",
		"PUT /experience/positions/{id}/achievements/order",
		"PATCH /experience/achievements/{id}",
		"DELETE /experience/achievements/{id}",
	)
	handlerstest.RejectsMalformedBody(t, r,
		"POST /experience/positions",
		"PUT /experience/positions/order",
		"PATCH /experience/positions/{id}",
		"POST /experience/positions/{id}/achievements",
		"PATCH /experience/achievements/{id}",
	)
	handlerstest.RejectsBadPathID(t, r,
		"DELETE /experience/positions/{id}",
		"DELETE /experience/achievements/{id}",
	)
}

func TestExperienceJourney(t *testing.T) {
	r := newTestRouter()

	w := do(t, r, http.MethodPost, "/experience/positions", `{"employer":"Acme","title":"Engineer","startDate":"2020-01-01"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create position status = %d, body %s", w.Code, w.Body)
	}
	var p dto.Position
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}

	w = do(t, r, http.MethodPost, "/experience/positions/"+p.ID+"/achievements", `{"text":"cut latency"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create achievement status = %d, body %s", w.Code, w.Body)
	}
	var a dto.Achievement
	if err := json.Unmarshal(w.Body.Bytes(), &a); err != nil {
		t.Fatal(err)
	}

	w = do(t, r, http.MethodPatch, "/experience/achievements/"+a.ID, `{"text":"cut p99 latency"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("update achievement status = %d, body %s", w.Code, w.Body)
	}

	w = do(t, r, http.MethodPut, "/experience/positions/"+p.ID+"/achievements/order", `{"ids":["`+a.ID+`"]}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("reorder achievements status = %d, body %s", w.Code, w.Body)
	}

	w = do(t, r, http.MethodPut, "/experience/positions/order", `{"ids":["`+p.ID+`"]}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("reorder positions status = %d, body %s", w.Code, w.Body)
	}

	w = do(t, r, http.MethodGet, "/experience/", "")
	var got []dto.Position
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := []dto.Position{{
		ID: p.ID, Employer: "Acme", Title: "Engineer", StartDate: ptr("2020-01-01"),
		Achievements: []dto.Achievement{{ID: a.ID, PositionID: p.ID, Text: "cut p99 latency"}},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("GET /experience (-want +got):\n%s", diff)
	}

	if w = do(t, r, http.MethodDelete, "/experience/positions/"+p.ID, ""); w.Code != http.StatusNoContent {
		t.Fatalf("delete position status = %d", w.Code)
	}
	if w = do(t, r, http.MethodDelete, "/experience/achievements/"+a.ID, ""); w.Code != http.StatusNotFound {
		t.Fatalf("delete cascaded achievement status = %d, want 404", w.Code)
	}
}
