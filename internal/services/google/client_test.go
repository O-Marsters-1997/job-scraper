package google_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

type tokenStore struct {
	row   dto.GoogleToken
	saved *dto.UpsertGoogleTokenInput
}

func (s tokenStore) GetGoogleToken(context.Context, string) (dto.GoogleToken, error) {
	return s.row, nil
}
func (s tokenStore) UpsertGoogleToken(_ context.Context, in dto.UpsertGoogleTokenInput) error {
	if s.saved != nil {
		*s.saved = in
	}
	return nil
}
func (tokenStore) DeleteGoogleToken(context.Context, string) error { return nil }

func linkedClient(t *testing.T, rt http.RoundTripper) (context.Context, *google.Client) {
	t.Helper()
	cipher := identitytest.NewCipher(t)
	access, err := cipher.Encrypt("access")
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := cipher.Encrypt("refresh")
	if err != nil {
		t.Fatal(err)
	}
	store := tokenStore{row: dto.GoogleToken{
		AccessTokenEnc: access, RefreshTokenEnc: refresh, TokenType: "Bearer", Expiry: time.Now().Add(time.Hour),
	}}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: rt})
	return ctx, google.NewClient("id", "secret", "http://localhost/cb", store, cipher)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func decodeJSON(t *testing.T, raw string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	return v
}

const docTabsJSON = `{"tabs":[
  {"tabProperties":{"tabId":"t.0"},"documentTab":{"marker":"first"},
   "childTabs":[{"tabProperties":{"tabId":"t.child"},"documentTab":{"marker":"child"}}]},
  {"tabProperties":{"tabId":"t.1"},"documentTab":{"marker":"second"}}]}`

func TestGetDocument(t *testing.T) {
	var gotURL string
	ctx, client := linkedClient(t, identitytest.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return jsonResponse(http.StatusOK, docTabsJSON), nil
	}))

	tests := []struct {
		name       string
		tabID      string
		wantMarker string
	}{
		{"top-level tab", "t.1", "second"},
		{"child tab", "t.child", "child"},
		{"empty id selects first", "", "first"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := client.GetDocument(ctx, "u1", "doc1", tt.tabID)
			if err != nil {
				t.Fatalf("GetDocument: %v", err)
			}
			var got struct {
				DocumentTab struct {
					Marker string `json:"marker"`
				} `json:"documentTab"`
			}
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if got.DocumentTab.Marker != tt.wantMarker {
				t.Errorf("marker = %q, want %q", got.DocumentTab.Marker, tt.wantMarker)
			}
			if !strings.Contains(gotURL, "/documents/doc1") || !strings.Contains(gotURL, "includeTabsContent=true") {
				t.Errorf("url = %q", gotURL)
			}
		})
	}

	t.Run("unknown tab is not found", func(t *testing.T) {
		_, err := client.GetDocument(ctx, "u1", "doc1", "t.nope")
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("GetDocument(t.nope) error = %v, want not found", err)
		}
	})
}

func TestAuthURLWriteAddsDriveFileAndGrantedScopes(t *testing.T) {
	client := google.NewClient("id", "secret", "http://localhost/cb", tokenStore{}, nil)

	read, err := url.Parse(client.AuthURL("s", false))
	if err != nil {
		t.Fatal(err)
	}
	if got := read.Query().Get("scope"); got != google.DriveReadonlyScope {
		t.Errorf("read scope = %q", got)
	}
	if read.Query().Has("include_granted_scopes") {
		t.Error("read-only URL must not carry include_granted_scopes")
	}

	write, err := url.Parse(client.AuthURL("s", true))
	if err != nil {
		t.Fatal(err)
	}
	q := write.Query()
	if q.Get("scope") != google.DriveFileScope || q.Get("include_granted_scopes") != "true" {
		t.Errorf("write URL query = %v", q)
	}
}

func TestSaveTokenStoresGrantedScope(t *testing.T) {
	cipher := identitytest.NewCipher(t)
	both := google.DriveReadonlyScope + " " + google.DriveFileScope

	tests := []struct {
		name     string
		stored   string
		extra    map[string]any
		wantSave string
	}{
		{"scope from token response", "", map[string]any{"scope": both}, both},
		{"missing scope keeps stored", both, nil, both},
		{"missing scope and no row defaults read-only", "", nil, google.DriveReadonlyScope},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved dto.UpsertGoogleTokenInput
			client := google.NewClient("id", "secret", "http://localhost/cb",
				tokenStore{row: dto.GoogleToken{Scope: tt.stored}, saved: &saved}, cipher)
			tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r"}
			if tt.extra != nil {
				tok = tok.WithExtra(tt.extra)
			}
			if err := client.SaveToken(context.Background(), "u1", tok); err != nil {
				t.Fatal(err)
			}
			if saved.Scope != tt.wantSave {
				t.Errorf("Scope = %q, want %q", saved.Scope, tt.wantSave)
			}
		})
	}
}

func TestHasScope(t *testing.T) {
	client := google.NewClient("id", "secret", "http://localhost/cb",
		tokenStore{row: dto.GoogleToken{Scope: google.DriveReadonlyScope + " " + google.DriveFileScope}}, nil)
	if ok, err := client.HasScope(context.Background(), "u1", google.DriveFileScope); err != nil || !ok {
		t.Errorf("HasScope(drive.file) = %v, %v", ok, err)
	}
	if ok, _ := client.HasScope(context.Background(), "u1", "other"); ok {
		t.Error("HasScope(other) = true")
	}
}

type capturedRequest struct {
	method, target, body string
}

func recordingClient(t *testing.T, status int, respBody string) (context.Context, *google.Client, *capturedRequest) {
	t.Helper()
	var got capturedRequest
	ctx, client := linkedClient(t, identitytest.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		got = capturedRequest{r.Method, r.URL.String(), string(b)}
		return jsonResponse(status, respBody), nil
	}))
	return ctx, client, &got
}

func TestCopyFile(t *testing.T) {
	ctx, client, got := recordingClient(t, http.StatusOK, `{"id":"new-doc"}`)

	id, err := client.CopyFile(ctx, "u1", "src", "Tailored CV")
	if err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	if id != "new-doc" {
		t.Errorf("CopyFile id = %q, want new-doc", id)
	}
	if got.method != http.MethodPost || !strings.HasSuffix(got.target, "/files/src/copy") {
		t.Errorf("sent %s %s", got.method, got.target)
	}
	if diff := cmp.Diff(map[string]any{"name": "Tailored CV"}, decodeJSON(t, got.body)); diff != "" {
		t.Errorf("body (-want +got):\n%s", diff)
	}
}

func TestBatchUpdate(t *testing.T) {
	ctx, client, got := recordingClient(t, http.StatusOK, `{}`)

	err := client.BatchUpdate(ctx, "u1", "doc", []json.RawMessage{json.RawMessage(`{"deleteContentRange":{}}`)})
	if err != nil {
		t.Fatalf("BatchUpdate: %v", err)
	}
	if got.method != http.MethodPost || !strings.HasSuffix(got.target, "/documents/doc:batchUpdate") {
		t.Errorf("sent %s %s", got.method, got.target)
	}
	want := map[string]any{"requests": []any{map[string]any{"deleteContentRange": map[string]any{}}}}
	if diff := cmp.Diff(want, decodeJSON(t, got.body)); diff != "" {
		t.Errorf("body (-want +got):\n%s", diff)
	}
}

func TestDeleteFile(t *testing.T) {
	t.Run("sends delete for the file", func(t *testing.T) {
		ctx, client, got := recordingClient(t, http.StatusNoContent, "")
		if err := client.DeleteFile(ctx, "u1", "doc"); err != nil {
			t.Fatalf("DeleteFile: %v", err)
		}
		if got.method != http.MethodDelete || !strings.HasSuffix(got.target, "/files/doc") {
			t.Errorf("sent %s %s", got.method, got.target)
		}
	})

	t.Run("forbidden response is an error", func(t *testing.T) {
		ctx, client, _ := recordingClient(t, http.StatusForbidden, "")
		if err := client.DeleteFile(ctx, "u1", "doc"); err == nil {
			t.Error("DeleteFile on 403 = nil error")
		}
	})
}
