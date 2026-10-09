package identity_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
)

func newTestRouter(t *testing.T) chi.Router {
	t.Helper()
	t.Setenv("SESSION_SECRET", "test-secret")
	m := identity.Build(testDeps(t, identity.Deps{}))
	r := chi.NewRouter()
	m.PublicRoutes(r)
	r.Group(func(r chi.Router) {
		r.Use(m.Middleware())
		m.Routes(r)
	})
	return r
}

func serve(t *testing.T, r http.Handler, route, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	method, path, _ := strings.Cut(route, " ")
	req := handlerstest.Request(t, method, path, body)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func wantStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d: %s", w.Code, status, w.Body)
	}
}

func cookie(t *testing.T, w *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %s cookie set", name)
	return nil
}

func signup(t *testing.T, r chi.Router, username string) *http.Cookie {
	t.Helper()
	w := handlerstest.Serve(t, r, "POST /auth/signup", `{"username":"`+username+`","password":"hunter2"}`)
	wantStatus(t, w, http.StatusCreated)
	return cookie(t, w, "session_id")
}

func startOAuth(t *testing.T, r chi.Router, query string) *httptest.ResponseRecorder {
	t.Helper()
	w := handlerstest.Serve(t, r, "GET /google/oauth/start?"+query, "")
	wantStatus(t, w, http.StatusTemporaryRedirect)
	return w
}

func oauthCallback(t *testing.T, r chi.Router, query string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	for _, c := range cookies {
		if state, _, ok := strings.Cut(c.Value, ":"); ok && c.Name == "oauth_state" {
			query = "state=" + state + "&" + query
		}
	}
	return serve(t, r, "GET /google/oauth/callback?"+query, "", cookies...)
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	handlerstest.RequiresAuth(t, newTestRouter(t),
		"POST /auth/logout",
		"GET /auth/me",
		"GET /profile",
		"PUT /profile",
		"GET /ai-prefs",
		"PUT /ai-credentials",
		"GET /google/oauth/callback",
		"GET /google/status",
		"DELETE /google/link",
	)

	m := identity.Build(testDeps(t, identity.Deps{}))
	noAuth := chi.NewRouter()
	m.PublicRoutes(noAuth)
	m.Routes(noAuth)
	handlerstest.RejectsMalformedBody(t, noAuth,
		"POST /auth/login", "POST /auth/signup", "PUT /profile", "PUT /ai-credentials",
	)
}

func TestSessionCookieJourney(t *testing.T) {
	r := newTestRouter(t)

	w := handlerstest.Serve(t, r, "POST /auth/signup", `{"username":"alice","password":"hunter2"}`)
	wantStatus(t, w, http.StatusCreated)
	session := cookie(t, w, "session_id")
	if session.Value == "" {
		t.Fatal("signup: want a session_id cookie")
	}
	if got := handlerstest.DecodeJSON[dto.AuthUserView](t, w.Body.Bytes()); got.Username != "alice" {
		t.Errorf("signup body = %+v, want username alice", got)
	}

	w = handlerstest.Serve(t, r, "POST /auth/login", `{"username":"alice","password":"hunter2"}`)
	wantStatus(t, w, http.StatusOK)
	if cookie(t, w, "session_id").Value == "" {
		t.Error("login: want a session_id cookie")
	}

	w = serve(t, r, "GET /auth/me", "", session)
	wantStatus(t, w, http.StatusOK)
	if got := handlerstest.DecodeJSON[dto.MeView](t, w.Body.Bytes()); got.Username != "alice" || got.IsAdmin {
		t.Errorf("me body = %+v, want username alice", got)
	}

	w = serve(t, r, "POST /auth/logout", "", session)
	wantStatus(t, w, http.StatusNoContent)
	if cleared := cookie(t, w, "session_id"); cleared.MaxAge >= 0 {
		t.Errorf("logout cookie MaxAge = %d, want negative", cleared.MaxAge)
	}

	wantStatus(t, serve(t, r, "GET /auth/me", "", session), http.StatusUnauthorized)
}

func TestProfileAndAIPrefsRoutes(t *testing.T) {
	r := newTestRouter(t)
	session := signup(t, r, "erin")

	wantStatus(t, serve(t, r, "PUT /profile", `{"email":"new@example.com"}`, session), http.StatusNoContent)
	w := serve(t, r, "GET /profile", "", session)
	wantStatus(t, w, http.StatusOK)
	if got := handlerstest.DecodeJSON[dto.Profile](t, w.Body.Bytes()); got.Email != "new@example.com" {
		t.Errorf("profile = %+v, want email new@example.com", got)
	}

	wantStatus(t, serve(t, r, "PUT /ai-credentials", `{"provider":"anthropic","apiKey":"sk-test"}`, session), http.StatusNoContent)
	w = serve(t, r, "GET /ai-prefs", "", session)
	wantStatus(t, w, http.StatusOK)
	got := handlerstest.DecodeJSON[dto.AIPrefsView](t, w.Body.Bytes())
	if len(got.ConfiguredProviders) != 1 || got.ConfiguredProviders[0] != "anthropic" {
		t.Errorf("ai-prefs = %+v, want [anthropic]", got)
	}
}

func TestGoogleOAuthStart(t *testing.T) {
	r := newTestRouter(t)

	w := startOAuth(t, r, "")
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "accounts.google.com") {
		t.Errorf("Location = %q, want a Google auth URL", loc)
	}
	if strings.Contains(loc, "drive.file") {
		t.Errorf("plain start Location = %q, must not request write scope", loc)
	}
	cookie(t, w, "oauth_state")

	w = startOAuth(t, r, "write=1")
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "include_granted_scopes=true") {
		t.Errorf("write start Location = %q, want write consent URL", loc)
	}
}

func TestGoogleOAuthConnectJourney(t *testing.T) {
	r := newTestRouter(t)
	session := signup(t, r, "carol")
	googleStatus := func() dto.GoogleStatus {
		w := serve(t, r, "GET /google/status", "", session)
		wantStatus(t, w, http.StatusOK)
		return handlerstest.DecodeJSON[dto.GoogleStatus](t, w.Body.Bytes())
	}

	start := startOAuth(t, r, "")
	wantStatus(t, oauthCallback(t, r, "code=auth-code", session, cookie(t, start, "oauth_state")), http.StatusTemporaryRedirect)
	if !googleStatus().Connected {
		t.Fatal("status after callback = disconnected, want connected")
	}

	wantStatus(t, serve(t, r, "DELETE /google/link", "", session), http.StatusNoContent)
	if googleStatus().Connected {
		t.Error("status after unlink = connected, want disconnected")
	}
}

func TestGoogleOAuthCallbackReturnPath(t *testing.T) {
	tests := []struct {
		name, ret, want string
	}{
		{"tailor route", "/jobs/42/tailor", "/jobs/42/tailor"},
		{"no return", "", "/settings/integrations"},
		{"external URL", "https://evil.example/x", "/settings/integrations"},
		{"protocol-relative", "//evil.example", "/settings/integrations"},
		{"unlisted in-app path", "/profile", "/settings/integrations"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRouter(t)
			session := signup(t, r, "dave")
			start := startOAuth(t, r, "write=1&return="+url.QueryEscape(tt.ret))

			w := oauthCallback(t, r, "code=c", append(start.Result().Cookies(), session)...)
			if got := w.Header().Get("Location"); got != tt.want {
				t.Errorf("Location = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("forged return cookie", func(t *testing.T) {
		r := newTestRouter(t)
		session := signup(t, r, "erin")
		state := cookie(t, startOAuth(t, r, ""), "oauth_state")
		forged := &http.Cookie{Name: "oauth_return", Value: "https://evil.example"}

		w := oauthCallback(t, r, "code=c", session, state, forged)
		if got := w.Header().Get("Location"); got != "/settings/integrations" {
			t.Errorf("Location = %q, want /settings/integrations", got)
		}
	})
}
