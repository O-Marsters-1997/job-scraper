package google_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
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

func newCipher(t *testing.T) *tokencrypt.Cipher {
	t.Helper()
	c, err := tokencrypt.New(base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012")))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

const docTabsJSON = `{"tabs":[
  {"tabProperties":{"tabId":"t.0"},"documentTab":{"marker":"first"},
   "childTabs":[{"tabProperties":{"tabId":"t.child"},"documentTab":{"marker":"child"}}]},
  {"tabProperties":{"tabId":"t.1"},"documentTab":{"marker":"second"}}]}`

func TestGetDocument(t *testing.T) {
	cipher := newCipher(t)
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

	var gotURL string
	transport := identitytest.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(docTabsJSON))}, nil
	})
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: transport})
	client := google.NewClient("id", "secret", "http://localhost/cb", store, cipher)

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
		if status, _ := apperr.StatusFor(err); status != http.StatusNotFound {
			t.Errorf("err = %v, want not found", err)
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
	cipher := newCipher(t)
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

func TestDriveWriteMethods(t *testing.T) {
	cipher := newCipher(t)
	access, _ := cipher.Encrypt("access")
	refresh, _ := cipher.Encrypt("refresh")
	store := tokenStore{row: dto.GoogleToken{
		AccessTokenEnc: access, RefreshTokenEnc: refresh, TokenType: "Bearer", Expiry: time.Now().Add(time.Hour),
	}}

	var method, target, body string
	status := http.StatusOK
	respBody := `{"id":"new-doc"}`
	transport := identitytest.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		method, target = r.Method, r.URL.String()
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(respBody))}, nil
	})
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: transport})
	client := google.NewClient("id", "secret", "http://localhost/cb", store, cipher)

	id, err := client.CopyFile(ctx, "u1", "src", "Tailored CV")
	if err != nil || id != "new-doc" {
		t.Fatalf("CopyFile = %q, %v", id, err)
	}
	if method != http.MethodPost || !strings.HasSuffix(target, "/files/src/copy") || body != `{"name":"Tailored CV"}` {
		t.Errorf("CopyFile sent %s %s %s", method, target, body)
	}

	err = client.BatchUpdate(ctx, "u1", "doc", []json.RawMessage{json.RawMessage(`{"deleteContentRange":{}}`)})
	if err != nil {
		t.Fatalf("BatchUpdate: %v", err)
	}
	if !strings.HasSuffix(target, "/documents/doc:batchUpdate") || body != `{"requests":[{"deleteContentRange":{}}]}` {
		t.Errorf("BatchUpdate sent %s %s", target, body)
	}

	status, respBody = http.StatusNoContent, ""
	if err := client.DeleteFile(ctx, "u1", "doc"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if method != http.MethodDelete || !strings.HasSuffix(target, "/files/doc") {
		t.Errorf("DeleteFile sent %s %s", method, target)
	}

	status = http.StatusForbidden
	if err := client.DeleteFile(ctx, "u1", "doc"); err == nil {
		t.Error("DeleteFile on 403 = nil error")
	}
}
