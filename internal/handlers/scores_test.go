package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/db"
)

type stubScoreEffects struct{ user string }

func (s *stubScoreEffects) GetScoringStatus(_ context.Context, userID string) (db.ScoringStatus, error) {
	s.user = userID
	return db.ScoringStatus{Pending: 2, Failed: 1, Stale: 3}, nil
}
func (s *stubScoreEffects) QueueRescore(_ context.Context, userID string) (int64, error) {
	s.user = userID
	return 4, nil
}

func TestScoresHandler_UsesAuthenticatedUser(t *testing.T) {
	store := &stubScoreEffects{}
	h := NewScoresHandler(store)
	status := httptest.NewRecorder()
	h.Status(status, withSession(httptest.NewRequest(http.MethodGet, "/scores/status", nil), "user-a"))
	if status.Code != http.StatusOK || store.user != "user-a" {
		t.Fatalf("status = %d user = %q", status.Code, store.user)
	}
	var payload db.ScoringStatus
	if err := json.NewDecoder(status.Body).Decode(&payload); err != nil || payload.Stale != 3 {
		t.Fatalf("status payload = %+v, %v", payload, err)
	}
	rescore := httptest.NewRecorder()
	h.Rescore(rescore, withSession(httptest.NewRequest(http.MethodPost, "/scores/rescore", nil), "user-b"))
	if rescore.Code != http.StatusOK || store.user != "user-b" {
		t.Fatalf("rescore = %d user = %q", rescore.Code, store.user)
	}
}
