package notify_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
)

func TestNtfyPusher_Send(t *testing.T) {
	t.Parallel()
	msg := dto.PushMessage{Title: "New job: Go Engineer", Body: "acme", URL: "fasttrack://jobs/j1?n=n1"}

	t.Run("posts title, click, bearer token and body", func(t *testing.T) {
		t.Parallel()
		var got http.Header
		var body string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r.Header.Clone()
			b, _ := io.ReadAll(r.Body)
			body = string(b)
		}))
		t.Cleanup(srv.Close)

		if err := notify.NewNtfyPusher(srv.URL, "tok").Send(t.Context(), msg); err != nil {
			t.Fatalf("Send() = %v", err)
		}
		want := map[string]string{"Title": msg.Title, "Click": msg.URL, "Authorization": "Bearer tok"}
		have := map[string]string{"Title": got.Get("Title"), "Click": got.Get("Click"), "Authorization": got.Get("Authorization")}
		if diff := cmp.Diff(want, have); diff != "" {
			t.Errorf("headers (-want +got):\n%s", diff)
		}
		if body != "acme" {
			t.Errorf("body = %q, want %q", body, "acme")
		}
	})

	t.Run("non-2xx is an error", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		t.Cleanup(srv.Close)

		if err := notify.NewNtfyPusher(srv.URL, "").Send(t.Context(), msg); err == nil {
			t.Error("Send() = nil, want error")
		}
	})
}
