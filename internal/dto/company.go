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
	Kind              CompanyKind

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
	CompanySortOpenJobs     CompanySort = "open_jobs"
)

type CompaniesQuery struct {
	Limit     string `json:"limit"`
	Offset    string `json:"offset"`
	Q         string `json:"q"`
	Tracked   string `json:"tracked"`
	Favourite string `json:"favourite"`
	NoBoard   string `json:"no_board"`
	Sort      string `json:"sort"`
	Poster    string `json:"poster"`
	ATS       string `json:"ats"`
}

type CompanyPageOptions struct {
	Limit         int32
	Offset        int32
	Search        string
	TrackedOnly   bool
	FavouriteOnly bool
	NoBoardOnly   bool
	Sort          CompanySort
	Poster        Poster // empty means all
	ATSSources    []string
}

type ATSCount struct {
	Source    string `json:"source"`
	Companies int    `json:"companies"`
}

type CompanyPage struct {
	Items []Company `json:"items"`
	Total int       `json:"total"`

	AllCompanies           int `json:"allCompanies"`
	VerifiedBoardCompanies int `json:"verifiedBoardCompanies"`
}

type CompanyTracking struct {
	CompanyID            string
	UserID               string
	ReviewState          string
	Enabled              bool
	CheckIntervalMinutes int
}
