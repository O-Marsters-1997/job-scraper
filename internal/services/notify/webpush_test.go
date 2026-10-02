package notify_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
)

func TestWebPusherSend(t *testing.T) {
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	browserKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	authSecret := make([]byte, 16)
	if _, err := rand.Read(authSecret); err != nil {
		t.Fatal(err)
	}
	pusher := notify.NewWebPusher(public, private, "mailto:owner@example.com")

	sendTo := func(t *testing.T, status int) error {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
		t.Cleanup(srv.Close)
		sub := dto.PushSubscriptionInput{
			Endpoint: srv.URL,
			Keys: dto.PushKeys{
				P256dh: base64.RawURLEncoding.EncodeToString(browserKey.PublicKey().Bytes()),
				Auth:   base64.RawURLEncoding.EncodeToString(authSecret),
			},
		}
		return pusher.Send(t.Context(), sub, dto.PushMessage{Title: "t", Body: "b", URL: "/jobs/1", Tag: "x"})
	}

	t.Run("201 is delivered", func(t *testing.T) {
		if err := sendTo(t, http.StatusCreated); err != nil {
			t.Fatalf("Send() = %v, want nil", err)
		}
	})
	for _, status := range []int{http.StatusNotFound, http.StatusGone} {
		t.Run(http.StatusText(status)+" maps to ErrSubscriptionGone", func(t *testing.T) {
			if err := sendTo(t, status); !errors.Is(err, notify.ErrSubscriptionGone) {
				t.Fatalf("Send() = %v, want ErrSubscriptionGone", err)
			}
		})
	}
	t.Run("500 is a plain error", func(t *testing.T) {
		err := sendTo(t, http.StatusInternalServerError)
		if err == nil || errors.Is(err, notify.ErrSubscriptionGone) {
			t.Fatalf("Send() = %v, want a non-gone error", err)
		}
	})
}
