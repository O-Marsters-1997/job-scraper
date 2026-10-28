package lineedit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

const chatCompletionsURL = "https://openrouter.ai/api/v1/chat/completions"

type OpenRouter struct {
	http *http.Client
	url  string
}

func NewOpenRouter() *OpenRouter {
	return NewOpenRouterAt(chatCompletionsURL, &http.Client{Timeout: 3 * time.Minute})
}

func NewOpenRouterAt(url string, hc *http.Client) *OpenRouter {
	return &OpenRouter{http: hc, url: url}
}

func (o *OpenRouter) Complete(ctx context.Context, apiKey string, p Prompt, onDelta func(string)) (Completion, error) {
	req := openrouter.Request{
		Model: p.Model,
		Messages: []openrouter.Message{
			{Role: "system", Content: p.System},
			{Role: "user", Content: p.User},
		},
		Usage: &openrouter.UsageOptions{Include: true},
	}
	reply, err := openrouter.ChatStream(ctx, o.http, o.url, apiKey, req, onDelta)
	if err != nil {
		return Completion{Cost: reply.Cost}, fmt.Errorf("complete: %w", err)
	}
	return Completion{Text: reply.Content, Cost: reply.Cost}, nil
}

func modelError(err error) error {
	if se, ok := errors.AsType[*openrouter.StatusError](err); ok {
		switch se.Code {
		case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden:
			return apperr.Unprocessable("OpenRouter refused the request; check your key and credit in Settings, AI")
		}
	}
	return apperr.Upstream("the suggestion model failed")
}
