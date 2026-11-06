package dto

type PromptPart struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

type PromptDocument struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

type PromptLab struct {
	Generation string           `json:"generation"`
	Main       string           `json:"main"`
	Parts      []PromptPart     `json:"parts"`
	Documents  []PromptDocument `json:"documents"`
	Checks     []string         `json:"checks"`
}

type PromptLabQuery struct {
	ID string `path:"id"`
}
