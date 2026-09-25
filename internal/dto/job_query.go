package dto

// JobsQuery is the query-string shape for GET /jobs. Fields stay strings so
// the generic query decoder can fill them from the URL; the jobs service
// parses and validates Limit, Availability and Cursor.
type JobsQuery struct {
	Limit        string `json:"limit"`
	Availability string `json:"availability"`
	CompanyID    string `json:"company_id"`
	Cursor       string `json:"cursor"`
}

// ResolveBoardQuery is the query-string shape for GET /sources/resolve.
type ResolveBoardQuery struct {
	URL string `json:"url"`
}

// ResolvedBoard is the response for GET /sources/resolve.
type ResolvedBoard struct {
	Source string `json:"source"`
	Value  string `json:"value"`
}
