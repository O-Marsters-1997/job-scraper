// Package wttj tracks which jobs and companies Welcome to the Jungle lists in its public sitemap.
package wttj

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

const (
	sitemapURL = "https://app.welcometothejungle.com/sitemap1.xml"
	userAgent  = "Mozilla/5.0 (compatible; job-scraper/1.0)"
)

var ErrBlocked = errors.New("wttj_blocked")

type Syncer interface {
	SyncWTTJSitemap(ctx context.Context, jobIDs, companyNames []string) (dto.SitemapDiff, error)
}

type Tracker struct {
	sync   Syncer
	client *http.Client
	url    string
}

func NewTracker(sync Syncer) *Tracker {
	return &Tracker{
		sync:   sync,
		client: &http.Client{Timeout: 2 * time.Minute, Transport: &http.Transport{Proxy: nil}},
		url:    sitemapURL,
	}
}

// RunOnce fetches the sitemap and records the diff. A 403 or 429 returns ErrBlocked,
// leaving the next scheduled run to try again.
func (t *Tracker) RunOnce(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url, nil)
	if err != nil {
		return fmt.Errorf("wttj sitemap: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("wttj sitemap: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusTooManyRequests:
		slog.WarnContext(ctx, "wttj sitemap blocked, pausing until next run", slog.String("status", resp.Status))
		return fmt.Errorf("wttj sitemap: status %s: %w", resp.Status, ErrBlocked)
	default:
		return fmt.Errorf("wttj sitemap: status %s", resp.Status)
	}

	jobIDs, companyNames, err := parseSitemap(resp.Body)
	if err != nil {
		return err
	}
	diff, err := t.sync.SyncWTTJSitemap(ctx, jobIDs, companyNames)
	if err != nil {
		return fmt.Errorf("wttj sitemap: %w", err)
	}
	if diff.GoneSkipped {
		slog.WarnContext(ctx, "wttj sitemap holds under half the live jobs, not closing any", slog.Int("total", len(jobIDs)))
	}
	slog.InfoContext(ctx, "wttj sitemap synced",
		slog.Int("new", len(diff.New)), slog.Int("gone", len(diff.Gone)), slog.Int(logger.KeyCount, len(jobIDs)))
	return nil
}
