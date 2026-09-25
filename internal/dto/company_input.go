package dto

type CreateCompanyInput struct {
	URL   string `json:"url"`
	Track *bool  `json:"track"`
}

type SetCompanyTrackingInput struct {
	Enabled              *bool `json:"enabled"`
	CheckIntervalMinutes *int  `json:"check_interval_minutes"`
}

type AddCompanyBoardInput struct {
	URL     string `json:"url"`
	Confirm bool   `json:"confirm"`
}
