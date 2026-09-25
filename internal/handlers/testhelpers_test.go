package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func withSession(r *http.Request, userID string) *http.Request {
	ctx := auth.WithSession(r.Context(), dto.Session{UserID: userID})
	return r.WithContext(ctx)
}

// withRouteID sets one or more chi URL params as alternating name, value
// pairs, e.g. withRouteID(r, "docId", "d1", "tabId", "t1").
func withRouteID(r *http.Request, pairs ...string) *http.Request {
	rctx := chi.NewRouteContext()
	for i := 0; i+1 < len(pairs); i += 2 {
		rctx.URLParams.Add(pairs[i], pairs[i+1])
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
