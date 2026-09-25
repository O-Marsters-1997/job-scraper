package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/sources/registry"
)

// Sources and ResolveBoard stay misfits per ADR 0020 (query-param driven,
// no per-request state), so they're plain functions rather than methods.

func Sources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(registry.Sources())
}

// ResolveBoard turns a direct ATS board URL into a {source, value} pair the client
// can use to pre-fill the add-target form. Returns 422 if no ATS board is recognised.
func ResolveBoard(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}
	source, token, ok := detect.ResolveBoard(rawURL)
	if !ok {
		http.Error(w, "could not resolve an ATS board from that URL", http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Source string `json:"source"`
		Value  string `json:"value"`
	}{Source: source, Value: token})
}
