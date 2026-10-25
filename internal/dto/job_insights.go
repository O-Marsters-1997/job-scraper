package dto

import "time"

type JobInsightsQuery struct {
	TZ string `json:"tz"`
}

type JobInsightsOptions struct {
	Now                  time.Time
	TZ                   string
	ExcludedCompanySlugs []string
}

// JobInsightGroup counts the jobs sharing one scrape day, source, score band and
// first finder. ScoreBand is -1 for an unscored job.
type JobInsightGroup struct {
	Day       string
	Source    string
	ScoreBand int
	FoundBy   string
	Jobs      int
	Scored    int
	ScoreSum  int
}

type JobInsightDay struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type JobInsightSource struct {
	Source   string `json:"source"`
	Count    int    `json:"count"`
	Scored   int    `json:"scored"`
	AvgScore int    `json:"avgScore"`
}

type JobInsightFinder struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type JobInsights struct {
	Total      int                `json:"total"`
	Scored     int                `json:"scored"`
	Daily      []JobInsightDay    `json:"daily"`
	Sources    []JobInsightSource `json:"sources"`
	ScoreBands []int              `json:"scoreBands"`
	FoundBy    []JobInsightFinder `json:"foundBy"`
}
