package dto

import "time"

type Job struct {
	ID                 string
	Title              string
	Location           string
	URL                string
	CompanySlug        string
	Source             string
	UpdatedAt          time.Time
	ScrapedAt          time.Time
	Description        string
	SalaryRaw          string
	WorkArrangement    string
	RelevanceScore     *int
	SuitabilityScore   *int
	Reasoning          *string
	Matched            []string
	Missing            []string
	SuitabilitySkipped bool
}
