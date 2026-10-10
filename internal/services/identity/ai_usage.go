package identity

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

const (
	openRouterKeyURL = "https://openrouter.ai/api/v1/key"
	aiUsageTTL       = 15 * time.Minute
	aiUsageProvider  = "openrouter"
)

// KeyUsageFetcher reads the spend and limit of one OpenRouter API key.
type KeyUsageFetcher interface {
	KeyUsage(ctx context.Context, apiKey string) (openrouter.KeyInfo, error)
}

type httpKeyUsage struct{ hc *http.Client }

func (f httpKeyUsage) KeyUsage(ctx context.Context, apiKey string) (openrouter.KeyInfo, error) {
	return openrouter.GetKey(ctx, f.hc, openRouterKeyURL, apiKey)
}

func newHTTPKeyUsage() KeyUsageFetcher {
	return httpKeyUsage{hc: &http.Client{Timeout: 10 * time.Second}}
}

type cachedUsage struct {
	view    dto.QuotaView
	expires time.Time
}

type aiUsageCache struct {
	mu          sync.Mutex
	entries     map[string]cachedUsage
	generations map[string]int
}

func (c *aiUsageCache) generation(userID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generations[userID]
}

func (c *aiUsageCache) get(userID string, now time.Time) (dto.QuotaView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[userID]
	if !ok || !now.Before(e.expires) {
		return dto.QuotaView{}, false
	}
	return e.view, true
}

func (c *aiUsageCache) put(userID string, generation int, view dto.QuotaView, expires time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generations[userID] != generation {
		return
	}
	if c.entries == nil {
		c.entries = make(map[string]cachedUsage)
	}
	c.entries[userID] = cachedUsage{view: view, expires: expires}
}

func (c *aiUsageCache) drop(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userID)
	if c.generations == nil {
		c.generations = make(map[string]int)
	}
	c.generations[userID]++
}

// AIUsage returns the spend and limit of userID's own OpenRouter key,
// cached for 15 minutes. Failures are reported in the view, not cached.
func (s *Service) AIUsage(ctx context.Context, userID string) (dto.QuotaView, error) {
	now := s.now()
	if view, ok := s.usage.get(userID, now); ok {
		return view, nil
	}
	generation := s.usage.generation(userID)
	view := dto.QuotaView{Provider: aiUsageProvider, Unit: "USD", Level: dto.UsageOK}

	apiKey, err := s.GetCredential(ctx, userID, aiUsageProvider)
	if errors.Is(err, data.ErrNotFound) {
		view.Status = dto.QuotaNotConfigured
		return view, nil
	}
	if err != nil {
		return dto.QuotaView{}, err
	}

	info, err := s.keyUsage.KeyUsage(ctx, apiKey)
	if err != nil {
		slog.WarnContext(ctx, "key usage failed", slog.String(logger.KeyProvider, aiUsageProvider), slog.Any(logger.KeyErr, err))
		view.Status = dto.QuotaError
		view.Error = "could not read usage from OpenRouter"
		return view, nil
	}

	view = keyInfoView(info, now)
	s.usage.put(userID, generation, view, now.Add(aiUsageTTL))
	return view, nil
}

func keyInfoView(info openrouter.KeyInfo, now time.Time) dto.QuotaView {
	view := dto.QuotaView{
		Provider:  aiUsageProvider,
		Status:    dto.QuotaOK,
		Unit:      "USD",
		Level:     dto.UsageOK,
		FetchedAt: &now,
	}
	if info.Limit == nil {
		spend := info.UsageMonthly
		view.Used = &spend
		return view
	}
	used := info.Usage
	if info.LimitRemaining != nil {
		used = max(0, *info.Limit-*info.LimitRemaining)
	}
	view.Used = &used
	view.Limit = info.Limit
	view.Percent, view.Level = dto.QuotaPercent(&used, info.Limit)
	if info.LimitReset != nil && *info.LimitReset == "monthly" {
		utc := now.UTC()
		resets := time.Date(utc.Year(), utc.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		view.ResetsAt = &resets
	}
	return view
}
