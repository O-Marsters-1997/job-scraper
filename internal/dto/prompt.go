package dto

type PromptPart struct {
	Path   string `json:"path"`
	Text   string `json:"text"`
	Source string `json:"source"` // "default" | "yours"
}

type PromptPartInput struct {
	Generation string `json:"-" path:"generation"`
	Path       string `json:"-" path:"path"`
	Text       string `json:"text"`
}

type PromptPartRef struct {
	Generation string `path:"generation"`
	Path       string `path:"path"`
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
