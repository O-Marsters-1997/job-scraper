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

type ImportSkill struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Exists   bool   `json:"exists"`
}

type ImportPreview struct {
	Positions []ImportPosition `json:"positions"`
	Skills    []ImportSkill    `json:"skills"`
}

type ImportInput struct {
	Positions []ImportPosition `json:"positions"`
	Skills    []ImportSkill    `json:"skills"`
}
