package handlers

import (
	"net/http"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

// ApplicationHandler holds the two routes driven by a query parameter
// (status_id, job_ids) rather than a path ID, so they stay misfits per ADR
// 0020 instead of binding through the generic adapter wrappers.
type ApplicationHandler struct {
	applications providers.ApplicationProvider
}

func NewApplicationHandler(applications providers.ApplicationProvider) *ApplicationHandler {
	return &ApplicationHandler{applications: applications}
}

func (h *ApplicationHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	userID, ok := Caller(w, r)
	if !ok {
		return
	}
	statusID := r.URL.Query().Get("status_id")

	var (
		apps any
		err  error
	)
	if statusID != "" {
		apps, err = h.applications.ListApplicationsByUserAndStatus(r.Context(), userID, statusID)
	} else {
		apps, err = h.applications.ListApplicationsByUser(r.Context(), userID)
	}
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, apps)
}

func (h *ApplicationHandler) GetApplicationsForJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := Caller(w, r)
	if !ok {
		return
	}
	raw := r.URL.Query().Get("job_ids")
	if raw == "" {
		WriteJSON(w, http.StatusOK, map[string]any{})
		return
	}
	jobIDs := strings.Split(raw, ",")
	m, err := h.applications.GetApplicationsForJobs(r.Context(), userID, jobIDs)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, m)
}
