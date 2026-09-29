package google_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

type tokenStore struct{ row dto.GoogleToken }

func (s tokenStore) GetGoogleToken(context.Context, string) (dto.GoogleToken, error) {
	return s.row, nil
}
func (tokenStore) UpsertGoogleToken(context.Context, dto.UpsertGoogleTokenInput) error { return nil }
func (tokenStore) DeleteGoogleToken(context.Context, string) error                     { return nil }

const docTabsJSON = `{"tabs":[
  {"tabProperties":{"tabId":"t.0"},"documentTab":{"marker":"first"},
   "childTabs":[{"tabProperties":{"tabId":"t.child"},"documentTab":{"marker":"child"}}]},
  {"tabProperties":{"tabId":"t.1"},"documentTab":{"marker":"second"}}]}`

func TestGetDocument(t *testing.T) {
	t.Setenv("GOOGLE_TOKEN_ENC_KEY", base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012")))
	access, err := tokencrypt.Encrypt("access")
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := tokencrypt.Encrypt("refresh")
	if err != nil {
		t.Fatal(err)
	}
	store := tokenStore{row: dto.GoogleToken{
		AccessTokenEnc: access, RefreshTokenEnc: refresh, TokenType: "Bearer", Expiry: time.Now().Add(time.Hour),
	}}

	var gotURL string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(docTabsJSON))}, nil
	})
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: transport})
	client := google.NewClient("id", "secret", "http://localhost/cb", store)

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
