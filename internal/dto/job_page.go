package dto

import "time"

type JobSort string

const (
	JobSortRelevant JobSort = "relevant"
	JobSortNewest   JobSort = "newest"
	JobSortBest     JobSort = "best"
)

type JobPageOptions struct {
	Limit  int32
	Offset int32
	Now    time.Time
	Sort   JobSort

	CompanyID      string
	ScoredOnly     bool
	MinSuitability *int32
	Graded         string
	Seen           string
	NoApplication  bool
	Poster         Poster // empty means all
	FavouriteOnly  bool
	Sources        []string
	Arrangements   []string
	SalaryOnly     bool
	SalaryMin      *float64
	SalaryMax      *float64
	Search         string
	Bands          []string
	ScrapedSince   *time.Time

	ExcludedCompanySlugs []string
}

type JobPage struct {
	Items []Job `json:"items"`
	Total int   `json:"total"`
}

type JobStatsOptions struct {
	Now                  time.Time
	DayStart             time.Time
	ExcludedCompanySlugs []string
}

type JobStats struct {
	Total        int `json:"total"`
	Sources      int `json:"sources"`
	NewToday     int `json:"newToday"`
	TodaySources int `json:"todaySources"`
}

type JobOverview struct {
	JobStats
	Top []Job `json:"top"`
}
