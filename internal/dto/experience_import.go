package dto

type ImportPreviewInput struct {
	DocID string `json:"docId"`
	TabID string `json:"tabId"`
}

type ImportPosition struct {
	Employer       string   `json:"employer"`
	Title          string   `json:"title"`
	StartDate      *string  `json:"startDate"`
	EndDate        *string  `json:"endDate"`
	Achievements   []string `json:"achievements"`
	EmployerExists bool     `json:"employerExists"`
}

type ImportPreview struct {
	Positions []ImportPosition `json:"positions"`
}

type ImportInput struct {
	Positions []ImportPosition `json:"positions"`
}
