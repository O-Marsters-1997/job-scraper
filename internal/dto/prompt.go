package dto

type PromptPart struct {
	Path    string  `json:"path"`
	Text    string  `json:"text"`
	Source  string  `json:"source"` // "default" | "yours"
	Default string  `json:"default"`
	BasedOn *string `json:"basedOn,omitempty"` // non-nil: the default moved since this was saved
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

type PromptTryInput struct {
	ID    string            `json:"-" path:"id"`
	Parts map[string]string `json:"parts"`
	Steer string            `json:"steer"`
}

type PromptTryDone struct {
	Text      string         `json:"text"`
	Findings  []DraftFinding `json:"findings"`
	Cost      float64        `json:"cost"`
	PromptKey string         `json:"promptKey"`
}
