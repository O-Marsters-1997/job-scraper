package dto

type JobsQuery struct {
	Limit        string `json:"limit"`
	Availability string `json:"availability"`
	CompanyID    string `json:"company_id"`
	Cursor       string `json:"cursor"`
}

type ResolveBoardQuery struct {
	URL string `json:"url"`
}

type ResolvedBoard struct {
	Source string `json:"source"`
	Value  string `json:"value"`
}
