package scraper_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
)

type redirectTransport struct {
	target *url.URL
}

func (t redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme = t.target.Scheme
	r.URL.Host = t.target.Host
	return http.DefaultTransport.RoundTrip(r)
}

func TestBuilderGreenhouseSource_FetchPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jobs":[{"id":1,"title":"Engineer","absolute_url":"https://boards.greenhouse.io/acme/jobs/1"}]}`))
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	src, ok := builder.BuildSource(dto.SourceTarget{Source: "greenhouse", Value: "acme", Enabled: true})
	if !ok {
		t.Fatal("expected greenhouse to build")
	}
	src.(interface{ Client() *http.Client }).Client().Transport = redirectTransport{target: serverURL}

	if _, _, err := src.FetchPage(t.Context(), ""); err != nil {
		t.Fatalf("FetchPage() = %v, want nil", err)
	}
}

func TestBuilderGreenhouseSource_FetchPageNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	src, ok := builder.BuildSource(dto.SourceTarget{Source: "greenhouse", Value: "acme", Enabled: true})
	if !ok {
		t.Fatal("expected greenhouse to build")
	}
	src.(interface{ Client() *http.Client }).Client().Transport = redirectTransport{target: serverURL}

	if _, _, err := src.FetchPage(t.Context(), ""); err == nil {
		t.Fatal("FetchPage() = nil, want error for 404")
	}
}

func TestVerifyBoardRejectsUnsupportedBoards(t *testing.T) {
	for _, pair := range [][2]string{{"missing", "acme"}, {"remoteok", "acme"}, {"greenhouse", ""}, {"greenhouse", "a/b"}} {
		if err := scraper.VerifyBoard(t.Context(), pair[0], pair[1]); err == nil {
			t.Errorf("accepted %q %q", pair[0], pair[1])
		}
	}
}
