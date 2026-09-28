package yc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/discover/yc"
)

func serve(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func TestHarvestFollowsPagination(t *testing.T) {
	body, err := os.ReadFile("testdata/page1.json")
	if err != nil {
		t.Fatal(err)
	}
	var page map[string]any
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page2" {
			_, _ = w.Write([]byte(`{"companies":[{"name":"Second Page Co","website":"https://second.example"}],"nextPage":""}`))
			return
		}
		page["nextPage"] = server.URL + "/page2"
		_ = json.NewEncoder(w).Encode(page)
	})

	companies, err := yc.New().WithBaseURL(server.URL + "/page1").Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(companies) != 26 {
		t.Fatalf("got %d companies, want 26", len(companies))
	}
	if first := companies[0]; first.Name != "Simantic" || first.Domain != "simantic.dev" {
		t.Errorf("first company = %+v, want Simantic / simantic.dev", first)
	}
	if last := companies[25]; last.Name != "Second Page Co" {
		t.Errorf("last company = %+v, want Second Page Co", last)
	}
}

func TestHarvestNormalisesDomains(t *testing.T) {
	server := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"companies":[
			{"name":"A","website":"https://www.acme.com/careers"},
			{"name":"B","website":"http://acme.com"},
			{"name":"C","website":"acme.com"},
			{"name":"D","website":""},
			{"name":"","website":""}
		],"nextPage":""}`))
	})

	got, err := yc.New().WithBaseURL(server.URL).Harvest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []discover.Company{{Name: "A", Domain: "acme.com"}, {Name: "B", Domain: "acme.com"}, {Name: "C", Domain: "acme.com"}, {Name: "D"}}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("company %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestHarvestErrors(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"invalid JSON":   func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not json")) },
		"non-200 status": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) },
	} {
		t.Run(name, func(t *testing.T) {
			server := serve(t, handler)
			if _, err := yc.New().WithBaseURL(server.URL).Harvest(t.Context()); err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}
