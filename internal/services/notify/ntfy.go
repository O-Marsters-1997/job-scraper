package notify

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type NtfyPusher struct {
	url, token string
	client     *http.Client
}

func NewNtfyPusher(url, token string) *NtfyPusher {
	return &NtfyPusher{url: url, token: token, client: &http.Client{Timeout: 10 * time.Second}}
}

// Send posts msg to the ntfy topic; msg.URL becomes the tap-through Click link.
func (p *NtfyPusher) Send(ctx context.Context, msg dto.PushMessage) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, strings.NewReader(msg.Body))
	if err != nil {
		return fmt.Errorf("ntfy build request: %w", err)
	}
	// ntfy decodes RFC 2047 header values, which keeps non-ASCII titles intact.
	req.Header.Set("Title", mime.QEncoding.Encode("utf-8", msg.Title))
	req.Header.Set("Click", msg.URL)
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("ntfy send: status %d", resp.StatusCode)
	}
	return nil
}
