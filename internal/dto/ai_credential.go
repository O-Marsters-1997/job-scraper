package dto

import (
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

type Provider string

const (
	ProviderOpenRouter Provider = "openrouter"
	ProviderAnthropic  Provider = "anthropic"
)

func ParseProvider(s string) (Provider, error) {
	switch p := Provider(s); p {
	case ProviderOpenRouter, ProviderAnthropic:
		return p, nil
	default:
		return "", apperr.Invalid(fmt.Sprintf("unknown provider %q", s))
	}
}

type UpsertCredentialInput struct {
	Provider Provider `json:"provider"`
	APIKey   *string  `json:"apiKey"`
}
