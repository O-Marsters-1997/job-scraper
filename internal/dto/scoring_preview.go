package dto

type ScoringPreviewInput struct {
	Preferences Preferences `json:"preferences"`
}

type ScoringPreview struct {
	DismissedInTop5 BeforeAfter    `json:"dismissedInTop5"`
	Jobs            []PreviewedJob `json:"jobs"`
}

type BeforeAfter struct {
	Before int `json:"before"`
	After  int `json:"after"`
}

type PreviewedJob struct {
	JobID     string     `json:"jobId"`
	Title     string     `json:"title"`
	Company   string     `json:"company"`
	Grade     string     `json:"grade,omitempty"`
	Before    int        `json:"before"`
	After     int        `json:"after"`
	AfterBand string     `json:"afterBand"`
	AfterRows []ScoreRow `json:"afterRows"`
}
