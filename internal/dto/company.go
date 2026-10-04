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

type CompaniesQuery struct {
	Limit     string `json:"limit"`
	Cursor    string `json:"cursor"`
	Q         string `json:"q"`
	Tracked   string `json:"tracked"`
	Favourite string `json:"favourite"`
}

type CompanyPageOptions struct {
	Limit         int32
	CursorName    string
	CursorID      string
	Search        string
	TrackedOnly   bool
	FavouriteOnly bool
}

type CompanyPage struct {
	Items      []Company `json:"items"`
	NextCursor string    `json:"next_cursor"`
}

type CompanyTracking struct {
	CompanyID            string
	UserID               string
	ReviewState          string
	Enabled              bool
	CheckIntervalMinutes int
}
