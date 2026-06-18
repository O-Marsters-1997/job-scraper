package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type APIPublisher struct {
	baseURL        string
	token          string
	client         *http.Client
	initialBackoff time.Duration // overridable in tests
}

func NewAPIPublisher(baseURL, token string) *APIPublisher {
	return &APIPublisher{
		baseURL:        baseURL,
		token:          token,
		client:         &http.Client{Timeout: 30 * time.Second},
		initialBackoff: time.Second,
	}
}

func (p *APIPublisher) Publish(ctx context.Context, jobs []dto.Job) error {
	for _, job := range jobs {
		if err := p.publishOne(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

func (p *APIPublisher) publishOne(ctx context.Context, job dto.Job) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("api_publisher: marshal: %w", err)
	}

	const maxRetries = 2
	backoff := p.initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/ingest", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("api_publisher: new request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if p.token != "" {
			req.Header.Set("Authorization", "Bearer "+p.token)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("api_publisher: request: %w", err)
			}
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return fmt.Errorf("api_publisher: permanent error %d for %s", resp.StatusCode, job.URL)
		}
		// 5xx: retry
		if attempt == maxRetries {
			return fmt.Errorf("api_publisher: server error %d for %s after %d retries", resp.StatusCode, job.URL, maxRetries)
		}
	}
	return nil
}
