package dto

type CreateSourceTargetInput struct {
	Source    string            `json:"source"`
	Value     string            `json:"value"`
	Enabled   *bool             `json:"enabled"`
	Filters   map[string]string `json:"filters"`
	ScrapeNow bool              `json:"scrape_now"`
	RunWindow RunWindowInput    `json:"run_window"`
}

type UpdateSourceTargetInput struct {
	ID        string         `json:"-" path:"id"`
	Enabled   *bool          `json:"enabled"`
	RunWindow RunWindowInput `json:"run_window"`
}
