package identity_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

func newTestRouter(t *testing.T, st *identitytest.FakeStore, gc *fakeGoogleClient) (chi.Router, *identity.Module) {
	t.Helper()
	m := buildModule(t, st, gc)
	r := chi.NewRouter()
	m.PublicRoutes(r)
	r.Group(func(r chi.Router) {
		r.Use(m.Middleware())
		m.Routes(r)
	})
	return r, m
}

func newRoutesOnlyRouter(t *testing.T, st *identitytest.FakeStore, gc *fakeGoogleClient) chi.Router {
	t.Helper()
	m := buildModule(t, st, gc)
	r := chi.NewRouter()
	m.PublicRoutes(r)
	m.Routes(r)
	return r
}

func jsonRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func signup(t *testing.T, r chi.Router, username string) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, jsonRequest(http.MethodPost, "/auth/signup", `{"username":"`+username+`","password":"hunter2"}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("signup status = %d: %s", w.Code, w.Body)
	}
	return sessionCookie(t, w)
}

func TestRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	authed, _ := newTestRouter(t, identitytest.NewFakeStore(), newFakeGoogleClient())
	handlerstest.RequiresAuth(t, authed,
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

	noAuth := newRoutesOnlyRouter(t, identitytest.NewFakeStore(), newFakeGoogleClient())
	handlerstest.RejectsMalformedBody(t, noAuth,
		"POST /auth/login", "POST /auth/signup", "PUT /profile", "PUT /ai-credentials",
	)
}

func TestSignupAndLoginSetTheSessionCookie(t *testing.T) {
	r, _ := newTestRouter(t, identitytest.NewFakeStore(), newFakeGoogleClient())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, jsonRequest(http.MethodPost, "/auth/signup", `{"username":"alice","password":"hunter2"}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("signup status = %d: %s", w.Code, w.Body)
	}
	if sessionCookie(t, w).Value == "" {
		t.Fatal("signup: want a session_id cookie")
	}
	var signedUp dto.AuthUserView
	if err := json.Unmarshal(w.Body.Bytes(), &signedUp); err != nil {
		t.Fatal(err)
	}
	if signedUp.Username != "alice" {
		t.Fatalf("signup body = %+v, want username alice", signedUp)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, jsonRequest(http.MethodPost, "/auth/login", `{"username":"alice","password":"hunter2"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d: %s", w.Code, w.Body)
	}
	if sessionCookie(t, w).Value == "" {
		t.Fatal("login: want a session_id cookie")
	}
}

func TestSessionCookieAuthenticatesProtectedRoutes(t *testing.T) {
	r, _ := newTestRouter(t, identitytest.NewFakeStore(), newFakeGoogleClient())
	cookie := signup(t, r, "bob")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", http.NoBody)
	req.AddCookie(cookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me status = %d: %s", w.Code, w.Body)
	}
	var me dto.MeView
	if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.Username != "bob" {
		t.Fatalf("me body = %+v, want username bob", me)
	}

	w = httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", http.NoBody)
	logoutReq.AddCookie(cookie)
	r.ServeHTTP(w, logoutReq)
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d: %s", w.Code, w.Body)
	}
	if cleared := sessionCookie(t, w); cleared.MaxAge >= 0 {
		t.Fatalf("logout cookie MaxAge = %d, want negative", cleared.MaxAge)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/auth/me", http.NoBody)
	req.AddCookie(cookie)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want 401", w.Code)
	}
}

func TestProfileAndAIPrefsRoutes(t *testing.T) {
	r, _ := newTestRouter(t, identitytest.NewFakeStore(), newFakeGoogleClient())
	cookie := signup(t, r, "erin")

	authedReq := func(method, path, body string) *http.Request {
		req := jsonRequest(method, path, body)
		req.AddCookie(cookie)
		return req
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authedReq(http.MethodPut, "/profile", `{"email":"new@example.com"}`))
	if w.Code != http.StatusNoContent {
		t.Fatalf("update profile status = %d: %s", w.Code, w.Body)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, authedReq(http.MethodGet, "/profile", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("get profile status = %d: %s", w.Code, w.Body)
	}
	var profile dto.ProfileView
	if err := json.Unmarshal(w.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Email != "new@example.com" {
		t.Fatalf("profile = %+v, want email new@example.com", profile)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, authedReq(http.MethodPut, "/ai-credentials", `{"provider":"anthropic","apiKey":"sk-test"}`))
	if w.Code != http.StatusNoContent {
		t.Fatalf("update ai-credentials status = %d: %s", w.Code, w.Body)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, authedReq(http.MethodGet, "/ai-prefs", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("get ai-prefs status = %d: %s", w.Code, w.Body)
	}
	var prefs dto.AIPrefsView
	if err := json.Unmarshal(w.Body.Bytes(), &prefs); err != nil {
		t.Fatal(err)
	}
	if len(prefs.ConfiguredProviders) != 1 || prefs.ConfiguredProviders[0] != "anthropic" {
		t.Fatalf("ai-prefs = %+v, want [anthropic]", prefs)
	}
}

func TestGoogleOAuthStartSetsStateCookieAndRedirects(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")
	r, _ := newTestRouter(t, identitytest.NewFakeStore(), newFakeGoogleClient())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/google/oauth/start", http.NoBody))
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "accounts.google.com") {
		t.Fatalf("Location = %q, want a Google auth URL", loc)
	}
	if stateCookie(t, w) == nil {
		t.Fatal("want an oauth_state cookie")
	}
}

func TestGoogleOAuthCallbackConnectsAndStatusReflectsIt(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")
	r, _ := newTestRouter(t, identitytest.NewFakeStore(), newFakeGoogleClient())
	session := signup(t, r, "carol")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/google/oauth/start", http.NoBody))
	state := stateCookie(t, w)

	w = httptest.NewRecorder()
	callbackReq := httptest.NewRequest(http.MethodGet, "/google/oauth/callback?state="+extractState(t, state.Value)+"&code=auth-code", http.NoBody)
	callbackReq.AddCookie(session)
	callbackReq.AddCookie(state)
	r.ServeHTTP(w, callbackReq)
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("callback status = %d: %s", w.Code, w.Body)
	}

	getStatus := func() dto.GoogleStatus {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/google/status", http.NoBody)
		req.AddCookie(session)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body)
		}
		var status dto.GoogleStatus
		if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		return status
	}

	if status := getStatus(); !status.Connected {
		t.Fatalf("status = %+v, want connected", status)
	}

	w = httptest.NewRecorder()
	linkReq := httptest.NewRequest(http.MethodDelete, "/google/link", http.NoBody)
	linkReq.AddCookie(session)
	r.ServeHTTP(w, linkReq)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete link status = %d: %s", w.Code, w.Body)
	}

	if status := getStatus(); status.Connected {
		t.Fatal("status after unlink = connected, want disconnected")
	}
}

func sessionCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_id" {
			return c
		}
	}
	t.Fatal("no session_id cookie set")
	return nil
}

func stateCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == "oauth_state" {
			return c
		}
	}
	return nil
}

var stateValuePattern = regexp.MustCompile(`^([0-9a-f]+):`)

func extractState(t *testing.T, signed string) string {
	t.Helper()
	m := stateValuePattern.FindStringSubmatch(signed)
	if m == nil {
		t.Fatalf("oauth_state cookie %q doesn't look like state:signature", signed)
	}
	return m[1]
}
