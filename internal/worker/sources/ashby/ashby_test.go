package ashby

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

// board_askdragonfly.json is a real capture from the Ashby posting-api
// (api.ashbyhq.com/posting-api/job-board/askdragonfly), trimmed to two jobs
// with shortened descriptions. Keep it real-shaped: it exists to catch API
// field drift, which a hand-written fixture cannot.
func TestParse_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "board_askdragonfly.json", "askdragonfly", parse)
}
