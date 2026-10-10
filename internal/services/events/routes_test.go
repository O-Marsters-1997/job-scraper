package events_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/events"
	"github.com/ollymarsters/job-scraper/internal/services/events/eventstest"
)

func TestRoutes(t *testing.T) {
	st := eventstest.NewFakeStore()
	r := chi.NewRouter()
	events.Build(events.Deps{Store: st, Snapshots: eventstest.Unscored()}).Routes(r)

	handlerstest.RequiresAuth(t, r, "POST /events")
	handlerstest.RejectsMalformedBody(t, r, "POST /events")

	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "POST /events", `{"type":"job_opened","subject_id":"job-1"}`)

	got, err := st.ListEvents(t.Context(), handlerstest.UserID, events.JobOpened)
	if err != nil || len(got) != 1 {
		t.Errorf("ListEvents() = %+v, %v, want the posted job_opened", got, err)
	}
}
