package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
)

func serve(method, pattern string, h http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Method(method, pattern, h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func authed(method, target, body string) *http.Request {
	return handlerstest.Authed(httptest.NewRequest(method, target, strings.NewReader(body)))
}

func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("body %q is not JSON: %v", w.Body.String(), err)
	}
	return v
}

func TestHandle(t *testing.T) {
	respond := func(w http.ResponseWriter, _ *http.Request, res string) {
		handlers.WriteJSON(w, http.StatusOK, res)
	}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	t.Run("decode error is written and call never runs", func(t *testing.T) {
		h := handlers.Handle(
			func(*http.Request) (string, error) { return "", apperr.Invalid("bad request") },
			func(context.Context, string) (string, error) {
				t.Error("call ran after a decode error")
				return "", nil
			},
			respond,
		)
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("call error is written", func(t *testing.T) {
		h := handlers.Handle(
			func(*http.Request) (string, error) { return "in", nil },
			func(context.Context, string) (string, error) { return "", errors.New("boom") },
			respond,
		)
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("decoded request reaches call and its result reaches respond", func(t *testing.T) {
		h := handlers.Handle(
			func(*http.Request) (string, error) { return "in", nil },
			func(_ context.Context, in string) (string, error) { return in + ":out", nil },
			respond,
		)
		w := httptest.NewRecorder()
		h(w, req)
		if got := decodeJSON[string](t, w); got != "in:out" {
			t.Fatalf("body = %q, want %q", got, "in:out")
		}
	})
}

func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name:       "kinded error maps to its status and message",
			err:        apperr.Conflict("board belongs to another company"),
			wantStatus: http.StatusConflict,
			wantBody:   map[string]any{"error": "board belongs to another company"},
		},
		{
			name:       "unkinded error hides its message and returns 500",
			err:        errors.New("pq: connection refused on 10.0.0.5:5432"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   map[string]any{"error": "internal server error"},
		},
		{
			name:       "fields attached to an error are merged into the body",
			err:        apperr.WithFields(apperr.Conflict("status is in use"), map[string]any{"count": float64(3)}),
			wantStatus: http.StatusConflict,
			wantBody:   map[string]any{"error": "status is in use", "count": float64(3)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.GetAll(func(context.Context, string) (string, error) { return "", tt.err })
			w := serve(http.MethodGet, "/x", h, authed(http.MethodGet, "/x", ""))
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if diff := cmp.Diff(tt.wantBody, decodeJSON[map[string]any](t, w)); diff != "" {
				t.Errorf("body (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetAll(t *testing.T) {
	t.Run("no session returns 401", func(t *testing.T) {
		h := handlers.GetAll(func(context.Context, string) (string, error) {
			t.Error("fn ran without a session")
			return "", nil
		})
		w := serve(http.MethodGet, "/x", h, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("nil slice is written as an empty array", func(t *testing.T) {
		h := handlers.GetAll(func(context.Context, string) ([]string, error) { return nil, nil })
		w := serve(http.MethodGet, "/x", h, authed(http.MethodGet, "/x", ""))
		got := decodeJSON[[]string](t, w)
		if got == nil || len(got) != 0 {
			t.Fatalf("body = %q, want []", w.Body.String())
		}
	})
}

func TestGetByID(t *testing.T) {
	h := handlers.GetByID(func(_ context.Context, userID, id string) (string, error) {
		return userID + ":" + id, nil
	})
	w := serve(http.MethodGet, "/x/{id}", h, authed(http.MethodGet, "/x/abc-123", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got, want := decodeJSON[string](t, w), handlerstest.UserID+":abc-123"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestQuery(t *testing.T) {
	type filter struct {
		StatusID string `json:"status_id"`
	}
	h := handlers.Query(func(_ context.Context, _ string, f filter) (filter, error) { return f, nil })

	t.Run("URL params decode into the input", func(t *testing.T) {
		w := serve(http.MethodGet, "/x", h, authed(http.MethodGet, "/x?status_id=s1", ""))
		if got := decodeJSON[filter](t, w); got.StatusID != "s1" {
			t.Fatalf("StatusID = %q, want s1", got.StatusID)
		}
	})

	t.Run("missing param decodes to the zero value", func(t *testing.T) {
		w := serve(http.MethodGet, "/x", h, authed(http.MethodGet, "/x", ""))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if got := decodeJSON[filter](t, w); got.StatusID != "" {
			t.Fatalf("StatusID = %q, want empty", got.StatusID)
		}
	})
}

func TestCreate(t *testing.T) {
	type in struct {
		ID   string `json:"id" path:"id"`
		Name string `json:"name"`
	}
	echo := handlers.Create(func(_ context.Context, _ string, body in) (in, error) { return body, nil })

	t.Run("malformed body returns 400 with a JSON error", func(t *testing.T) {
		w := serve(http.MethodPost, "/x/{id}", echo, authed(http.MethodPost, "/x/1", "not json"))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if _, ok := decodeJSON[map[string]any](t, w)["error"]; !ok {
			t.Errorf("body %q has no error field", w.Body.String())
		}
	})

	t.Run("empty body decodes to the zero value", func(t *testing.T) {
		w := serve(http.MethodPost, "/x/{id}", echo, authed(http.MethodPost, "/x/1", ""))
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
		}
		if diff := cmp.Diff(in{ID: "1"}, decodeJSON[in](t, w)); diff != "" {
			t.Errorf("body (-want +got):\n%s", diff)
		}
	})

	t.Run("path tag wins over the body", func(t *testing.T) {
		w := serve(http.MethodPost, "/x/{id}", echo, authed(http.MethodPost, "/x/real-id", `{"id":"spoofed","name":"a"}`))
		want := in{ID: "real-id", Name: "a"}
		if diff := cmp.Diff(want, decodeJSON[in](t, w)); diff != "" {
			t.Errorf("body (-want +got):\n%s", diff)
		}
	})

	t.Run("several path params are filled", func(t *testing.T) {
		type twoIDs struct {
			DocID string `json:"docId" path:"docId"`
			TabID string `json:"tabId" path:"tabId"`
		}
		h := handlers.Create(func(_ context.Context, _ string, body twoIDs) (twoIDs, error) { return body, nil })
		w := serve(http.MethodPost, "/x/{docId}/tabs/{tabId}", h, authed(http.MethodPost, "/x/d1/tabs/t1", ""))
		if diff := cmp.Diff(twoIDs{DocID: "d1", TabID: "t1"}, decodeJSON[twoIDs](t, w)); diff != "" {
			t.Errorf("body (-want +got):\n%s", diff)
		}
	})

	t.Run("struct{} output writes 204 with no body", func(t *testing.T) {
		h := handlers.Create(func(context.Context, string, struct{}) (struct{}, error) { return struct{}{}, nil })
		w := serve(http.MethodPost, "/x", h, authed(http.MethodPost, "/x", ""))
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
		}
		if w.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", w.Body.String())
		}
	})
}

func TestUpdate(t *testing.T) {
	type in struct {
		ID    string `json:"id" path:"id"`
		Notes string `json:"notes"`
	}
	h := handlers.Update(func(_ context.Context, _ string, body in) (in, error) { return body, nil })
	w := serve(http.MethodPatch, "/x/{id}", h, authed(http.MethodPatch, "/x/app-1", `{"notes":"followed up"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if diff := cmp.Diff(in{ID: "app-1", Notes: "followed up"}, decodeJSON[in](t, w)); diff != "" {
		t.Errorf("body (-want +got):\n%s", diff)
	}
}

func TestDelete(t *testing.T) {
	var gotID string
	h := handlers.Delete(func(_ context.Context, _, id string) error {
		gotID = id
		return nil
	})
	w := serve(http.MethodDelete, "/x/{id}", h, authed(http.MethodDelete, "/x/abc-123", ""))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if gotID != "abc-123" {
		t.Fatalf("id = %q, want abc-123", gotID)
	}
}
