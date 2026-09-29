package cvtailor_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
)

func TestTailorRoutesRejectUnauthedAndMalformedRequests(t *testing.T) {
	r := newTestRouter()
	handlerstest.RequiresAuth(t, r,
		"GET /tailoring/cvs/{docId}/{tabId}/headings",
		"PUT /tailoring/cvs/{docId}/{tabId}/headings",
		"GET /tailoring/jobs/{jobId}/suggestions",
	)
	handlerstest.RejectsMalformedBody(t, r, "PUT /tailoring/cvs/{docId}/{tabId}/headings")
}
