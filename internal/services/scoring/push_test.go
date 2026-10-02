package scoring_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

type fakePusher struct {
	gone  map[string]bool
	sent  []string
	other error
}

func (p *fakePusher) Send(_ context.Context, sub dto.PushSubscriptionInput, _ dto.PushMessage) error {
	p.sent = append(p.sent, sub.Endpoint)
	if p.gone[sub.Endpoint] {
		return notify.ErrSubscriptionGone
	}
	if sub.Endpoint == "https://push.example/flaky" {
		return p.other
	}
	return nil
}

func pushRouter(t *testing.T, st scoring.Store, pusher scoring.PushSender, vapidKey string) chi.Router {
	t.Helper()
	deps := newDeps(t, st)
	deps.Pusher = pusher
	deps.VAPIDPublicKey = vapidKey
	r := chi.NewRouter()
	scoring.Build(deps).Routes(r)
	return r
}

func subscribeBody(endpoint string) string {
	return `{"endpoint":"` + endpoint + `","keys":{"p256dh":"p","auth":"a"}}`
}

func TestPushRoutesRequireAuth(t *testing.T) {
	r := pushRouter(t, newFakeStore(), nil, "key")
	handlerstest.RequiresAuth(t, r,
		"GET /push/vapid-public-key", "POST /push/subscriptions", "DELETE /push/subscriptions", "POST /push/test")
	handlerstest.RejectsMalformedBody(t, r, "POST /push/subscriptions", "DELETE /push/subscriptions")
}

func TestVAPIDPublicKey(t *testing.T) {
	t.Run("returns the configured key", func(t *testing.T) {
		r := pushRouter(t, newFakeStore(), nil, "pub-key")
		got := handlerstest.Do[dto.VAPIDKey](t, r, http.StatusOK, "GET /push/vapid-public-key", "")
		if got.Key != "pub-key" {
			t.Errorf("GET /push/vapid-public-key key = %q, want pub-key", got.Key)
		}
	})
	t.Run("is 404 when push is unconfigured", func(t *testing.T) {
		r := pushRouter(t, newFakeStore(), nil, "")
		if rec := handlerstest.Serve(t, r, "GET /push/vapid-public-key", ""); rec.Code != http.StatusNotFound {
			t.Errorf("GET /push/vapid-public-key = %d, want 404", rec.Code)
		}
	})
}

func TestPushSubscriptions(t *testing.T) {
	t.Run("resubscribing the same endpoint keeps one row", func(t *testing.T) {
		st := newFakeStore()
		r := pushRouter(t, st, nil, "key")
		for range 2 {
			if rec := handlerstest.Serve(t, r, "POST /push/subscriptions", subscribeBody("https://push.example/a")); rec.Code != http.StatusNoContent {
				t.Fatalf("POST /push/subscriptions = %d, want 204", rec.Code)
			}
		}
		subs, _ := st.ListPushSubscriptions(t.Context(), handlerstest.UserID)
		if len(subs) != 1 {
			t.Errorf("subscriptions = %v, want exactly one", subs)
		}
	})
	t.Run("rejects a subscription with no keys", func(t *testing.T) {
		r := pushRouter(t, newFakeStore(), nil, "key")
		if rec := handlerstest.Serve(t, r, "POST /push/subscriptions", `{"endpoint":"https://push.example/a"}`); rec.Code != http.StatusBadRequest {
			t.Errorf("POST /push/subscriptions = %d, want 400", rec.Code)
		}
	})
	t.Run("unsubscribe is idempotent", func(t *testing.T) {
		st := newFakeStore()
		r := pushRouter(t, st, nil, "key")
		handlerstest.Serve(t, r, "POST /push/subscriptions", subscribeBody("https://push.example/a"))
		for range 2 {
			if rec := handlerstest.Serve(t, r, "DELETE /push/subscriptions", `{"endpoint":"https://push.example/a"}`); rec.Code != http.StatusNoContent {
				t.Fatalf("DELETE /push/subscriptions = %d, want 204", rec.Code)
			}
		}
		if subs, _ := st.ListPushSubscriptions(t.Context(), handlerstest.UserID); len(subs) != 0 {
			t.Errorf("subscriptions = %v, want none", subs)
		}
	})
}

func TestPushTest(t *testing.T) {
	st := newFakeStore()
	pusher := &fakePusher{gone: map[string]bool{"https://push.example/gone": true}, other: context.DeadlineExceeded}
	r := pushRouter(t, st, pusher, "key")
	for _, ep := range []string{"https://push.example/ok", "https://push.example/gone", "https://push.example/flaky"} {
		handlerstest.Serve(t, r, "POST /push/subscriptions", subscribeBody(ep))
	}

	if rec := handlerstest.Serve(t, r, "POST /push/test", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("POST /push/test = %d, want 204", rec.Code)
	}

	wantSent := []string{"https://push.example/ok", "https://push.example/gone", "https://push.example/flaky"}
	if diff := cmp.Diff(wantSent, pusher.sent); diff != "" {
		t.Errorf("sent endpoints (-want +got):\n%s", diff)
	}
	subs, _ := st.ListPushSubscriptions(t.Context(), handlerstest.UserID)
	var left []string
	for _, s := range subs {
		left = append(left, s.Endpoint)
	}
	wantLeft := []string{"https://push.example/ok", "https://push.example/flaky"}
	if diff := cmp.Diff(wantLeft, left); diff != "" {
		t.Errorf("remaining subscriptions (-want +got):\n%s", diff)
	}
}
