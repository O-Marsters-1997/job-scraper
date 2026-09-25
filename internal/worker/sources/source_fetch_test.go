package sources_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func TestProtectedSourceDoesNotFetchDirectlyWithoutCredentials(t *testing.T) {
	t.Setenv("BRIGHTDATA_PROXY_URL", "")
	src := sources.NewBase(sources.Config{Name: "protected", UseProxy: true})
	if _, err := src.Get(context.Background(), "https://8.8.8.8/jobs"); err == nil {
		t.Fatal("protected fetch should fail without credentials")
	}
}

func TestFetcherRejectsUnsafeRedirect(t *testing.T) {
	src := sources.NewBase(sources.Config{Name: "direct"})
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/private", nil)
	if err := src.Client().CheckRedirect(req, nil); err == nil {
		t.Fatal("private redirect should be rejected")
	}
}
