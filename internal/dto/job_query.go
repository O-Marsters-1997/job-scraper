package dto

type JobsQuery struct {
	Page      string `json:"page"`
	PageSize  string `json:"page_size"`
	Sort      string `json:"sort"`
	CompanyID string `json:"company_id"`
	Scored    string `json:"scored"`
	Suit      string `json:"suit"`
	Graded    string `json:"graded"`
	Seen      string `json:"seen"`
	Poster    string `json:"poster"`
	Favourite string `json:"favourite"`
	Source    string `json:"source"`
	Work      string `json:"work"`
	Salary    string `json:"salary"`
	SalaryMin string `json:"salary_min"`
	SalaryMax string `json:"salary_max"`
	Q         string `json:"q"`
	SkipTotal string `json:"skip_total"`
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

type JobOverviewQuery struct {
	DayStart string `json:"day_start"`
	Ranked   string `json:"ranked"`
}

// TriageQuery's Poster is direct or agency; empty means direct.
type TriageQuery struct {
	Poster string `json:"poster"`
}
