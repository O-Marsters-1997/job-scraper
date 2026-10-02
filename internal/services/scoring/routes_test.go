package scoring_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

func newTestRouter(t *testing.T) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	scoring.Build(newDeps(t, newFakeStore())).Routes(r)
	return r
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newTestRouter(t)

	handlerstest.RequiresAuth(t, r,
		"GET /scoring-config",
		"PUT /scoring-config",
		"GET /scoring-options",
		"GET /scores/status",
		"POST /scores/recompute",
	)
	handlerstest.RejectsMalformedBody(t, r, "PUT /scoring-config")
}

func TestRoutesHappyPath(t *testing.T) {
	r := newTestRouter(t)

	options := handlerstest.Do[dto.ScoringOptionsView](t, r, http.StatusOK, "GET /scoring-options", "")
	if !slices.ContainsFunc(options.Options, func(o dto.ScoringOption) bool { return o.ID == "tech:go" }) {
		t.Errorf("GET /scoring-options options = %+v, want the seeded tech:go option", options.Options)
	}

	body := `{"notifyThreshold":80,"excludedTitleKeywords":[],"excludedCompanies":[],"excludedLocations":[],
		"preferences":{"picks":[{"optionId":"tech:go","stance":"nice"}]}}`
	updated := handlerstest.Do[dto.ScoringConfigView](t, r, http.StatusOK, "PUT /scoring-config", body)
	wantPicks := []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}}
	if updated.NotifyThreshold != 80 {
		t.Errorf("PUT /scoring-config notifyThreshold = %d, want 80", updated.NotifyThreshold)
	}
	if diff := cmp.Diff(wantPicks, updated.Preferences.Picks); diff != "" {
		t.Errorf("PUT /scoring-config picks (-want +got):\n%s", diff)
	}

	saved := handlerstest.Do[dto.ScoringConfigView](t, r, http.StatusOK, "GET /scoring-config", "")
	if saved.NotifyThreshold != 80 {
		t.Errorf("GET /scoring-config notifyThreshold = %d, want the saved 80", saved.NotifyThreshold)
	}
	if diff := cmp.Diff(wantPicks, saved.Preferences.Picks); diff != "" {
		t.Errorf("GET /scoring-config picks (-want +got):\n%s", diff)
	}

	status := handlerstest.Do[dto.ScoringStatus](t, r, http.StatusOK, "GET /scores/status", "")
	if status.Pending != 0 {
		t.Errorf("GET /scores/status pending = %d, want 0", status.Pending)
	}

	recomputed := handlerstest.Do[dto.RecomputeResult](t, r, http.StatusOK, "POST /scores/recompute", "")
	if recomputed.Recomputed != 0 {
		t.Errorf("POST /scores/recompute recomputed = %d, want 0 (no scored jobs)", recomputed.Recomputed)
	}
}

func TestFeedbackRoutes(t *testing.T) {
	r := newTestRouter(t)

	handlerstest.RequiresAuth(t, r, "POST /scoring-feedback/overall", "GET /scoring-feedback", "DELETE /scoring-feedback/{id}")
	handlerstest.RejectsMalformedBody(t, r, "POST /scoring-feedback/overall")

	blank := handlerstest.Serve(t, r, "POST /scoring-feedback/overall", `{"reason":" "}`)
	if blank.Code != http.StatusBadRequest {
		t.Errorf("POST /scoring-feedback/overall blank status = %d, want 400", blank.Code)
	}

	created := handlerstest.Do[dto.ScoreFeedback](t, r, http.StatusCreated, "POST /scoring-feedback/overall", `{"reason":"too generous"}`)
	if created.Kind != "overall" || created.Reason != "too generous" {
		t.Errorf("POST /scoring-feedback/overall = %+v, want an overall entry with the reason", created)
	}

	listed := handlerstest.Do[dto.ScoreFeedbackPage](t, r, http.StatusOK, "GET /scoring-feedback", "")
	if len(listed.Entries) != 1 || listed.Entries[0].ID != created.ID || listed.Total != 1 {
		t.Errorf("GET /scoring-feedback = %+v, want just the created entry, total 1", listed)
	}

	for _, target := range []string{"?kind=nonsense", "?page=0", "?page=x"} {
		if rec := handlerstest.Serve(t, r, "GET /scoring-feedback"+target, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("GET /scoring-feedback%s status = %d, want 400", target, rec.Code)
		}
	}
	jobs := handlerstest.Do[dto.ScoreFeedbackPage](t, r, http.StatusOK, "GET /scoring-feedback?kind=job", "")
	if len(jobs.Entries) != 0 || jobs.Total != 0 {
		t.Errorf("GET /scoring-feedback?kind=job = %+v, want empty", jobs)
	}

	if rec := handlerstest.Serve(t, r, "DELETE /scoring-feedback/nope", ""); rec.Code != http.StatusNotFound {
		t.Errorf("DELETE unknown id status = %d, want 404", rec.Code)
	}
	if rec := handlerstest.Serve(t, r, "DELETE /scoring-feedback/"+created.ID, ""); rec.Code != http.StatusNoContent {
		t.Errorf("DELETE own entry status = %d, want 204", rec.Code)
	}
}

func TestJobFeedbackRoute(t *testing.T) {
	st := newFakeStore()
	seedScoredJob(st, handlerstest.UserID, "tech:go")
	r := chi.NewRouter()
	scoring.Build(newDeps(t, st)).Routes(r)

	handlerstest.RequiresAuth(t, r, "POST /scoring-feedback/job")
	handlerstest.RejectsMalformedBody(t, r, "POST /scoring-feedback/job")

	for name, tc := range map[string]struct {
		body string
		want int
	}{
		"unknown job":   {`{"jobId":"nope","direction":"higher","reason":"r"}`, http.StatusNotFound},
		"bad direction": {`{"jobId":"job-1","direction":"up","reason":"r"}`, http.StatusBadRequest},
		"blank reason":  {`{"jobId":"job-1","direction":"higher","reason":" "}`, http.StatusBadRequest},
	} {
		if rec := handlerstest.Serve(t, r, "POST /scoring-feedback/job", tc.body); rec.Code != tc.want {
			t.Errorf("POST /scoring-feedback/job %s status = %d, want %d", name, rec.Code, tc.want)
		}
	}

	created := handlerstest.Do[dto.ScoreFeedback](t, r, http.StatusCreated, "POST /scoring-feedback/job", `{"jobId":"job-1","direction":"higher","reason":"too low"}`)
	if created.Kind != "job" || created.Snapshot.Score == nil || *created.Snapshot.Score != 72 || len(created.Snapshot.Options) != 1 {
		t.Errorf("POST /scoring-feedback/job = %+v, want a job entry with the frozen score and one Option", created)
	}
}

func TestCollectionFeedbackRoute(t *testing.T) {
	st := newFakeStore()
	seedScoredJob(st, handlerstest.UserID, "tech:go")
	r := chi.NewRouter()
	scoring.Build(newDeps(t, st)).Routes(r)

	handlerstest.RequiresAuth(t, r, "POST /scoring-feedback/collection")
	handlerstest.RejectsMalformedBody(t, r, "POST /scoring-feedback/collection")

	if rec := handlerstest.Serve(t, r, "POST /scoring-feedback/collection", `{"jobIds":[],"reason":"r"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("POST /scoring-feedback/collection with no ids status = %d, want 400", rec.Code)
	}

	created := handlerstest.Do[dto.ScoreFeedback](t, r, http.StatusCreated, "POST /scoring-feedback/collection", `{"jobIds":["job-1"],"filters":{"q":"go"},"reason":"off"}`)
	if created.Kind != "collection" || len(created.Snapshot.Ranking) != 1 || created.Snapshot.Ranking[0].Score == nil {
		t.Errorf("POST /scoring-feedback/collection = %+v, want a collection entry ranking the scored Job", created)
	}
}
