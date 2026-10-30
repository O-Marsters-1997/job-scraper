package openrouter

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

// Refusal returns an apperr.Unprocessable naming why OpenRouter rejected the
// key (401, 402 or 403), including its own error.message when the body
// parses. It returns nil for any other error.
func Refusal(err error) error {
	se, ok := errors.AsType[*StatusError](err)
	if !ok {
		return nil
	}
	var msg string
	switch se.Code {
	case http.StatusUnauthorized:
		msg = "OpenRouter rejected your API key; check it in Settings, AI"
	case http.StatusPaymentRequired:
		msg = "OpenRouter says your account is out of credit for this request; top up or lower the key's limit in Settings, AI"
	case http.StatusForbidden:
		msg = "OpenRouter refused the request (key limit exceeded or content moderated); check your key in Settings, AI"
	default:
		return nil
	}
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(se.Body), &body) == nil && body.Error.Message != "" {
		msg += ": " + body.Error.Message
	}
	return apperr.Unprocessable(msg)
}
