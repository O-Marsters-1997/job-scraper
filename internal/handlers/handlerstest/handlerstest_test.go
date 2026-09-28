package handlerstest_test

import (
	"context"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
)

type thing struct {
	ID   string `json:"id" path:"id"`
	Name string `json:"name"`
}

func toyRouter() chi.Router {
	things := map[string]thing{"thing-1": {ID: "thing-1", Name: "widget"}}

	r := chi.NewRouter()
	r.Get("/things/{id}", handlers.GetByID(func(_ context.Context, _, id string) (thing, error) {
		t, ok := things[id]
		if !ok {
			return thing{}, apperr.NotFound("thing not found")
		}
		return t, nil
	}))
	r.Post("/things", handlers.Create(func(_ context.Context, _ string, in thing) (thing, error) {
		return in, nil
	}))
	return r
}

func TestRequiresAuth(t *testing.T) {
	handlerstest.RequiresAuth(t, toyRouter(), "GET /things/{id}", "POST /things")
}

func TestRejectsMalformedBody(t *testing.T) {
	handlerstest.RejectsMalformedBody(t, toyRouter(), "POST /things")
}

func TestRejectsBadPathID(t *testing.T) {
	handlerstest.RejectsBadPathID(t, toyRouter(), "GET /things/{id}")
}
