package dto

type JobScore struct {
	JobID            string
	UserID           string
	RelevanceScore   *int
	SuitabilityScore *int
}
