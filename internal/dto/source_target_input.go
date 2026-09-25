package dto

type CreateSourceTargetInput struct {
	Source    string            `json:"source"`
	Value     string            `json:"value"`
	Enabled   *bool             `json:"enabled"`
	Filters   map[string]string `json:"filters"`
	ScrapeNow bool              `json:"scrape_now"`
}

type UpdateSourceTargetInput struct {
	ID                   string `json:"-" path:"id"`
	Enabled              *bool  `json:"enabled"`
	CheckIntervalMinutes *int   `json:"check_interval_minutes"`
}
