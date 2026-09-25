package dto

type UpsertCredentialInput struct {
	Provider string  `json:"provider"`
	APIKey   *string `json:"apiKey"`
}
