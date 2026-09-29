package dto

type JobsQuery struct {
	Limit        string `json:"limit"`
	Availability string `json:"availability"`
	CompanyID    string `json:"company_id"`
	Scored       string `json:"scored"`
	Cursor       string `json:"cursor"`
}

type ResolveBoardQuery struct {
	URL string `json:"url"`
}

type ResolvedURL struct {
	Kind    string            `json:"kind"`
	Source  string            `json:"source"`
	Value   string            `json:"value"`
	Filters map[string]string `json:"filters"`
	Dropped []string          `json:"dropped"`
	URL     string            `json:"url"`
}
