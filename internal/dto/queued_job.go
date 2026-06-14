package dto

type QueuedJob struct {
	URL       string
	Relevance int
	Card      Job
}
