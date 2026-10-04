package dto

import "time"

type Company struct {
	ID                string
	Slug              string
	Name              string
	ATSSource         string
	ATSToken          string
	Domain            string
	LinkedInCompanyID string
	LastCrawledAt     *time.Time
	FirstSeenAt       time.Time

	JobCount             int
	Tracked              bool
	Favourite            bool
	ReviewState          string
	TargetID             string
	CheckIntervalMinutes int
	LastCheckedAt        *time.Time
}

type CompanySort string

const (
	CompanySortRelevance    CompanySort = "relevance"
	CompanySortAlphabetical CompanySort = "alphabetical"
)

type CompaniesQuery struct {
	Limit     string `json:"limit"`
	Offset    string `json:"offset"`
	Q         string `json:"q"`
	Tracked   string `json:"tracked"`
	Favourite string `json:"favourite"`
	NoBoard   string `json:"no_board"`
	Sort      string `json:"sort"`
}

type CompanyPageOptions struct {
	Limit         int32
	Offset        int32
	Search        string
	TrackedOnly   bool
	FavouriteOnly bool
	NoBoardOnly   bool
	Sort          CompanySort
}

type CompanyPage struct {
	Items []Company `json:"items"`
	Total int       `json:"total"`
}

type CompanyTracking struct {
	CompanyID            string
	UserID               string
	ReviewState          string
	Enabled              bool
	CheckIntervalMinutes int
}
