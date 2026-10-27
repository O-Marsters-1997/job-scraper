package notify_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
)

type routeStore struct {
	fakeStore
	read []string
}

func (s *routeStore) ListNotifications(_ context.Context, userID string) ([]dto.Notification, error) {
	return []dto.Notification{{ID: "n1", JobID: "j1", Title: userID}}, nil
}

func (s *routeStore) MarkNotificationRead(_ context.Context, userID, id string) error {
	s.read = append(s.read, userID+"/"+id)
	return nil
}

func (s *routeStore) MarkAllNotificationsRead(_ context.Context, userID string) error {
	s.read = append(s.read, userID+"/all")
	return nil
}

func TestRoutes(t *testing.T) {
	t.Parallel()
	st := &routeStore{}
	r := chi.NewRouter()
	notify.Build(notify.Deps{Store: st}).Routes(r)

	handlerstest.RequiresAuth(t, r, "GET /notifications", "POST /notifications/{id}/read", "POST /notifications/read-all")

	got := handlerstest.Do[[]dto.Notification](t, r, http.StatusOK, "GET /notifications", "")
	if diff := cmp.Diff([]dto.Notification{{ID: "n1", JobID: "j1", Title: handlerstest.UserID}}, got); diff != "" {
		t.Errorf("GET /notifications (-want +got):\n%s", diff)
	}

	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "POST /notifications/abc/read", "")
	handlerstest.Do[struct{}](t, r, http.StatusNoContent, "POST /notifications/read-all", "")
	want := []string{handlerstest.UserID + "/abc", handlerstest.UserID + "/all"}
	if diff := cmp.Diff(want, st.read); diff != "" {
		t.Errorf("store calls (-want +got):\n%s", diff)
	}
}
