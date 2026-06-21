package dto

// ScrapeRequest is enqueued by the API when a user adds a source target with
// scrape_now:true. The worker consumer pops it and runs a one-off scrape for
// the target, bypassing the regular min-interval gate.
type ScrapeRequest struct {
	Target SourceTarget
}
