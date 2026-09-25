package dto

type AIPrefsView struct {
	SuitabilityModel    string   `json:"suitabilityModel"`
	ReasoningModel      string   `json:"reasoningModel"`
	AvailableModels     []string `json:"availableModels"`
	ConfiguredProviders []string `json:"configuredProviders"`
	ScoringEnabled      bool     `json:"scoringEnabled"`
}

type UpdateAIPrefsInput struct {
	SuitabilityModel string `json:"suitabilityModel"`
	ReasoningModel   string `json:"reasoningModel"`
}
