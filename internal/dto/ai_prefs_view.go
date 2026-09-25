package dto

type AIPrefsView struct {
	ConfiguredProviders []string `json:"configuredProviders"`
	ScoringEnabled      bool     `json:"scoringEnabled"`
}
