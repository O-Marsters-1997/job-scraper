package dto

// UpsertCredentialInput is the body for PUT /ai-credentials. A nil APIKey
// deletes the stored credential for Provider; otherwise the key is saved.
type UpsertCredentialInput struct {
	Provider string  `json:"provider"`
	APIKey   *string `json:"apiKey"`
}
