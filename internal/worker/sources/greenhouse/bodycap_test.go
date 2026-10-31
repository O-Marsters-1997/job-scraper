package greenhouse_test

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/greenhouse"
)

type oversizedBody struct{}

func (oversizedBody) Read([]byte) (int, error) { return 0, &http.MaxBytesError{Limit: 64 << 20} }

func TestNew_BodyCap(t *testing.T) {
	t.Parallel()
	if got := greenhouse.New("elastic").Cfg().MaxBodyBytes; got <= proxy.DefaultMaxBody {
		t.Errorf("Cfg().MaxBodyBytes = %d, want above the default %d", got, proxy.DefaultMaxBody)
	}
}

func TestPollBoard_OversizedBodyIsUnparseable(t *testing.T) {
	t.Parallel()
	src := greenhouse.New("elastic")
	src.Client().Transport = identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(oversizedBody{})}, nil
	})
	if _, err := src.PollBoard(t.Context()); !errors.Is(err, sources.ErrUnparseable) {
		t.Errorf("PollBoard() error = %v, want ErrUnparseable", err)
	}
}
