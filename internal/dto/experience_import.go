package dto

type ImportPreviewInput struct {
	DocID string `json:"docId"`
	TabID string `json:"tabId"`
}

// ImportPosition is one candidate Position from a CV Tab. Dates are
// YYYY-MM-DD, nil when the heading held none the parser could read.
type ImportPosition struct {
	Employer     string   `json:"employer"`
	Title        string   `json:"title"`
	StartDate    *string  `json:"startDate"`
	EndDate      *string  `json:"endDate"`
	Achievements []string `json:"achievements"`
	// EmployerExists is set by the preview when the Bank already holds a Position
	// at this employer; the commit ignores it.
	EmployerExists bool `json:"employerExists"`
}

type ImportPreview struct {
	Positions []ImportPosition `json:"positions"`
}

type ImportInput struct {
	Positions []ImportPosition `json:"positions"`
}
