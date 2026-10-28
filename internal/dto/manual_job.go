package dto

type AddManualJobInput struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Description string `json:"description"`
}

type ManualJobResult struct {
	Job      Job  `json:"job"`
	Existing bool `json:"existing"`
}
