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

type AddCompanyBoardInput struct {
	CompanyID string `json:"-" path:"id"`
	URL       string `json:"url"`
	Confirm   bool   `json:"confirm"`
}
