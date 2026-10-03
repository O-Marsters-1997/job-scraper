package dto

type CreateCompanyInput struct {
	URL   string `json:"url"`
	Track *bool  `json:"track"`
}

type SetCompanyTrackingInput struct {
	CompanyID            string `json:"-" path:"id"`
	Enabled              *bool  `json:"enabled"`
	CheckIntervalMinutes *int   `json:"check_interval_minutes"`
}

type SetCompanyReviewInput struct {
	CompanyID string `json:"-" path:"id"`
	State     string `json:"state"`
}

type ExcludeCompanyInput struct {
	CompanyID string `json:"-" path:"id"`
}

type UnexcludeCompanyInput struct {
	CompanyID string `json:"-" path:"id"`
	// RemoveName is the Added the exclusion reported: only a name that
	// action added is taken back out of ExcludedCompanies.
	RemoveName bool `json:"removeName"`
}

type CompanyExclusion struct {
	Name  string `json:"name"`
	Added bool   `json:"added"`
}

type AddCompanyBoardInput struct {
	CompanyID string `json:"-" path:"id"`
	URL       string `json:"url"`
	Confirm   bool   `json:"confirm"`
}
