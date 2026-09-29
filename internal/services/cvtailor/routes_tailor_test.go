package cvtailor_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
)

func TestTailorRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newTestRouter()
	handlerstest.RequiresAuth(t, r,
		"GET /tailoring/cvs/{docId}/{tabId}/headings",
		"PUT /tailoring/cvs/{docId}/{tabId}/headings",
		"GET /tailoring/jobs/{jobId}/suggestions",
		"POST /tailoring/drafts",
		"GET /tailoring/drafts/{id}",
	)
	handlerstest.RejectsMalformedBody(t, r, "PUT /tailoring/cvs/{docId}/{tabId}/headings", "POST /tailoring/drafts")
}

func TestDraftRoutesQueueAndReadADraft(t *testing.T) {
	r := newTestRouter()
	var p dto.Position
	var a dto.Achievement
	decode := func(w *httptest.ResponseRecorder, into any) {
		t.Helper()
		if err := json.Unmarshal(w.Body.Bytes(), into); err != nil {
			t.Fatalf("decode %s: %v", w.Body, err)
		}
	}
	decode(do(t, r, http.MethodPost, "/experience/positions", `{"employer":"Acme","title":"Engineer"}`), &p)
	decode(do(t, r, http.MethodPost, "/experience/positions/"+p.ID+"/achievements", `{"text":"cut latency"}`), &a)
	if w := do(t, r, http.MethodPut, "/tailoring/cvs/d/t/headings", `{"mappings":[{"headingText":"Acme","positionId":"`+p.ID+`"}]}`); w.Code != http.StatusOK {
		t.Fatalf("save headings status = %d, body %s", w.Code, w.Body)
	}

	w := do(t, r, http.MethodPost, "/tailoring/drafts", `{"jobId":"job-1","docId":"d","tabId":"t","achievementIds":["`+a.ID+`"]}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("create draft status = %d, body %s", w.Code, w.Body)
	}
	var ref dto.DraftRef
	decode(w, &ref)

	w = do(t, r, http.MethodGet, "/tailoring/drafts/"+ref.ID, "")
	var got dto.Draft
	decode(w, &got)
	if w.Code != http.StatusOK || got.ID != ref.ID || got.Status != "pending" || got.DraftDocURL != nil {
		t.Errorf("GET draft = %d %+v, want the pending Draft", w.Code, got)
	}
	if w = do(t, r, http.MethodGet, "/tailoring/drafts/missing", ""); w.Code != http.StatusNotFound {
		t.Errorf("GET unknown draft status = %d, want 404", w.Code)
	}
}
