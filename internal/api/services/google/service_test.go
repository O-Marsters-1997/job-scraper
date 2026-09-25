package google_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/api/services/google"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type fakeClient struct {
	httpClient  *http.Client
	httpErr     error
	deletedUser string
	exchangeErr error
	savedToken  *oauth2.Token
}

func (f *fakeClient) AuthURL(state string) string {
	return "https://accounts.google.com/o?state=" + state
}

func (f *fakeClient) Exchange(context.Context, string) (*oauth2.Token, error) {
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	return &oauth2.Token{AccessToken: "tok"}, nil
}

func (f *fakeClient) SaveToken(_ context.Context, _ string, tok *oauth2.Token) error {
	f.savedToken = tok
	return nil
}

func (f *fakeClient) HTTPClientForUser(context.Context, string) (*http.Client, error) {
	return f.httpClient, f.httpErr
}

func (f *fakeClient) DeleteToken(_ context.Context, userID string) error {
	f.deletedUser = userID
	return nil
}

func TestStatusNotConnected(t *testing.T) {
	svc := google.New(&fakeClient{httpErr: providers.ErrGoogleTokenNotFound})
	got, err := svc.Status(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Connected {
		t.Fatalf("got = %+v, want disconnected", got)
	}
}

func TestStatusUnusableTokenTreatedAsDisconnected(t *testing.T) {
	svc := google.New(&fakeClient{httpErr: providers.ErrGoogleTokenUnusable})
	got, err := svc.Status(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Connected {
		t.Fatalf("got = %+v, want disconnected", got)
	}
}

func TestStatusConnectedFetchesEmail(t *testing.T) {
	hc := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"email":"alice@example.com"}`)),
		}, nil
	})}
	svc := google.New(&fakeClient{httpClient: hc})
	got, err := svc.Status(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Connected || got.Email != "alice@example.com" {
		t.Fatalf("got = %+v", got)
	}
}

func TestConnectSavesTheExchangedToken(t *testing.T) {
	client := &fakeClient{}
	svc := google.New(client)
	if err := svc.Connect(context.Background(), "user-1", "code"); err != nil {
		t.Fatal(err)
	}
	if client.savedToken == nil || client.savedToken.AccessToken != "tok" {
		t.Fatalf("savedToken = %+v", client.savedToken)
	}
}

func TestConnectPropagatesExchangeError(t *testing.T) {
	want := errors.New("exchange failed")
	svc := google.New(&fakeClient{exchangeErr: want})
	if err := svc.Connect(context.Background(), "user-1", "code"); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestDisconnect(t *testing.T) {
	client := &fakeClient{}
	svc := google.New(client)
	if err := svc.Disconnect(context.Background(), "user-1", ""); err != nil {
		t.Fatal(err)
	}
	if client.deletedUser != "user-1" {
		t.Fatalf("deletedUser = %q, want user-1", client.deletedUser)
	}
}
