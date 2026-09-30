package handlerstest_test

import (
	"context"
	"net/http"
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

func TestDo(t *testing.T) {
	r := toyRouter()

	got := handlerstest.Do[thing](t, r, http.StatusOK, "GET /things/thing-1", "")
	if want := (thing{ID: "thing-1", Name: "widget"}); got != want {
		t.Errorf("Do(GET /things/thing-1) = %+v, want %+v", got, want)
	}

	created := handlerstest.Do[thing](t, r, http.StatusCreated, "POST /things", `{"id":"x","name":"gadget"}`)
	if created.Name != "gadget" {
		t.Errorf("Do(POST /things) name = %q, want gadget", created.Name)
	}

	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "GET /things/missing", "")
}

func TestServe(t *testing.T) {
	w := handlerstest.Serve(t, toyRouter(), "GET /things/thing-1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("Serve status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
}

func TestDecodeJSON(t *testing.T) {
	got := handlerstest.DecodeJSON[map[string]int](t, []byte(`{"a":1}`))
	if got["a"] != 1 {
		t.Errorf("DecodeJSON = %v, want a=1", got)
	}
}
