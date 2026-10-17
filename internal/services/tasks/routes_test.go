package tasks_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/tasks"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/taskstest"
)

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	tasks.Build(tasks.Deps{Store: taskstest.NewFakeStore()}).Routes(r)

	handlerstest.RequiresAuth(t, r, "GET /task-types/", "POST /task-types/", "PATCH /task-types/{id}", "DELETE /task-types/{id}",
		"GET /tasks/", "POST /tasks/", "PATCH /tasks/{id}", "PUT /tasks/{id}/done")
	handlerstest.RejectsMalformedBody(t, r, "POST /task-types/", "PATCH /task-types/{id}", "POST /tasks/", "PATCH /tasks/{id}", "PUT /tasks/{id}/done")
	handlerstest.RejectsBadPathID(t, r, "DELETE /task-types/{id}", "PUT /tasks/{id}/done")
}

func TestRoutesHappyPath(t *testing.T) {
	t.Parallel()
	store := taskstest.NewFakeStore()
	r := chi.NewRouter()
	m := tasks.Build(tasks.Deps{Store: store})
	m.Routes(r)
	if err := m.SeedDefaults(t.Context(), nil, handlerstest.UserID); err != nil {
		t.Fatal(err)
	}

	types := handlerstest.Do[[]dto.TaskType](t, r, http.StatusOK, "GET /task-types/", "")
	if len(types) != 4 {
		t.Fatalf("got %d task types, want 4", len(types))
	}
	offer := handlerstest.Do[dto.TaskType](t, r, http.StatusCreated, "POST /task-types/", `{"name":"Offer","colour":"#15803d"}`)
	handlerstest.Do[struct{}](t, r, http.StatusConflict, "POST /task-types/", `{"name":"Offer","colour":"#15803d"}`)
	renamed := handlerstest.Do[dto.TaskType](t, r, http.StatusOK, "PATCH /task-types/"+offer.ID, `{"name":"Offers","colour":"#000000"}`)
	if renamed.Name != "Offers" || renamed.Colour != "#000000" {
		t.Fatalf("renamed type = %+v, want Offers #000000", renamed)
	}
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "DELETE /task-types/"+offer.ID, "")
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "DELETE /task-types/"+offer.ID, "")

	created := handlerstest.Do[dto.Task](t, r, http.StatusCreated, "POST /tasks/",
		`{"type_id":"`+types[0].ID+`","link":"https://example.com/a","notes":"call"}`)
	if created.TypeName != types[0].Name || created.Link != "https://example.com/a" {
		t.Fatalf("created task = %+v, want type %q and the link", created, types[0].Name)
	}
	handlerstest.Do[struct{}](t, r, http.StatusBadRequest, "POST /tasks/",
		`{"type_id":"`+types[0].ID+`","link":"javascript:alert(1)"}`)

	edited := handlerstest.Do[dto.Task](t, r, http.StatusOK, "PATCH /tasks/"+created.ID,
		`{"type_id":"`+types[1].ID+`","link":"","notes":"edited"}`)
	if edited.TypeName != types[1].Name || edited.Link != "" || edited.Notes != "edited" {
		t.Fatalf("edited task = %+v, want type %q, no link and notes", edited, types[1].Name)
	}
	handlerstest.Do[struct{}](t, r, http.StatusBadRequest, "PATCH /tasks/"+created.ID,
		`{"type_id":"`+types[1].ID+`","link":"javascript:alert(1)"}`)
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "PATCH /tasks/"+created.ID,
		`{"type_id":"00000000-0000-0000-0000-00000000ffff"}`)
	handlerstest.Do[struct{}](t, r, http.StatusNotFound, "PATCH /tasks/does-not-exist",
		`{"type_id":"`+types[1].ID+`"}`)

	open := handlerstest.Do[dto.TaskPage](t, r, http.StatusOK, "GET /tasks/", "")
	if open.Total != 1 || len(open.Items) != 1 {
		t.Fatalf("open page = %+v, want one task", open)
	}

	done := handlerstest.Do[dto.Task](t, r, http.StatusOK, "PUT /tasks/"+created.ID+"/done", `{"done":true}`)
	if done.CompletedAt == nil {
		t.Fatal("PUT done: CompletedAt = nil, want set")
	}
	if got := handlerstest.Do[dto.TaskPage](t, r, http.StatusOK, "GET /tasks/", ""); got.Total != 0 || got.Items == nil {
		t.Fatalf("open page after done = %+v, want empty non-nil items", got)
	}
	if got := handlerstest.Do[dto.TaskPage](t, r, http.StatusOK, "GET /tasks/?done=true", ""); got.Total != 1 {
		t.Fatalf("done page = %+v, want one task", got)
	}
}
