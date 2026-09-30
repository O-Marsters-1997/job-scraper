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
