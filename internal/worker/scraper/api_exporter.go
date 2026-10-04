package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

type APIExporter struct {
	baseURL        string
	token          string
	client         *http.Client
	initialBackoff time.Duration
}

func NewAPIExporter(baseURL, token string) *APIExporter {
	client := &http.Client{Timeout: 30 * time.Second, Transport: logger.Transport(nil)}
	return &APIExporter{baseURL: baseURL, token: token, client: client, initialBackoff: time.Second}
}

func (p *APIExporter) WithInitialBackoff(d time.Duration) *APIExporter {
	p.initialBackoff = d
	return p
}

func (p *APIExporter) BulkExport(ctx context.Context, jobs []dto.Job) error {
	for len(jobs) > 0 {
		count := min(len(jobs), 100)
		var body []byte
		for {
			var err error
			body, err = json.Marshal(struct {
				Jobs []dto.Job `json:"jobs"`
			}{Jobs: jobs[:count]})
			if err != nil {
				return fmt.Errorf("api_exporter: marshal batch: %w", err)
			}
			if len(body) <= 2<<20 {
				break
			}
			if count == 1 {
				return fmt.Errorf("api_exporter: job exceeds 2 MiB")
			}
			count = max(1, count/2)
		}
		batch := jobs[:count]
		response, err := p.post(ctx, "/ingest/batch", body)
		if err != nil {
			return err
		}
		var result struct {
			Results []struct {
				Status string `json:"status"`
				Reason string `json:"reason"`
			} `json:"results"`
		}
		if err := json.Unmarshal(response, &result); err != nil || len(result.Results) != len(batch) {
			return fmt.Errorf("api_exporter: invalid batch response")
		}
		for idx, item := range result.Results {
			switch item.Status {
			case "new", "changed", "unchanged", "merged":
			case "rejected":
				return fmt.Errorf("api_exporter: job %s rejected: %s", batch[idx].URL, item.Reason)
			default:
				return fmt.Errorf("api_exporter: unknown outcome %q", item.Status)
			}
		}
		jobs = jobs[count:]
	}
	return nil
}

func (p *APIExporter) Export(ctx context.Context, job dto.Job) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("api_exporter: marshal: %w", err)
	}
	response, err := p.post(ctx, "/ingest", body)
	if err != nil {
		return err
	}
	var result struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("api_exporter: decode outcome: %w", err)
	}
	switch result.Status {
	case "new", "changed", "unchanged", "merged":
		return nil
	case "rejected":
		return fmt.Errorf("api_exporter: job %s rejected: %s", job.URL, result.Reason)
	default:
		return fmt.Errorf("api_exporter: unknown outcome %q", result.Status)
	}
}

func (p *APIExporter) post(ctx context.Context, path string, body []byte) ([]byte, error) {
	const maxRetries = 2
	backoff := p.initialBackoff
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("api_exporter: new request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if p.token != "" {
			req.Header.Set("Authorization", "Bearer "+p.token)
		}
		resp, err := p.client.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("api_exporter: request: %w", err)
			}
			continue
		}
		response, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("api_exporter: read response: %w", readErr)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return response, nil
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, fmt.Errorf("api_exporter: permanent error %d", resp.StatusCode)
		}
		if attempt == maxRetries {
			return nil, fmt.Errorf("api_exporter: server error %d after %d retries", resp.StatusCode, maxRetries)
		}
	}
	return nil, fmt.Errorf("api_exporter: exhausted retries")
}
