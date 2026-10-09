package jobsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

const (
	// DecodoSubscriptionsURL lists the account's proxy subscriptions.
	DecodoSubscriptionsURL = "https://api.decodo.com/v2/subscriptions"
	// ProxyUsageInterval is how often the API refreshes the Decodo snapshot.
	ProxyUsageInterval = 15 * time.Minute

	proxyProvider = "decodo"
	proxyUnit     = "GB"
)

// ProxyQuota is one proxy subscription's traffic in GB and its reset time.
type ProxyQuota struct {
	Used     float64
	Limit    float64
	ResetsAt time.Time
}

// QuotaFetcher reads the current proxy quota from the provider.
type QuotaFetcher interface {
	FetchQuota(ctx context.Context) (ProxyQuota, error)
}

type decodoQuota struct {
	hc     *http.Client
	url    string
	apiKey string
}

// NewDecodoQuota reads the first Decodo subscription from url using apiKey.
func NewDecodoQuota(hc *http.Client, url, apiKey string) QuotaFetcher {
	return decodoQuota{hc: hc, url: url, apiKey: apiKey}
}

type decodoSubscription struct {
	TrafficPerPeriod string `json:"traffic_per_period"`
	TrafficLimit     string `json:"traffic_limit"`
	ValidUntil       string `json:"valid_until"`
}

func (d decodoQuota) FetchQuota(ctx context.Context) (ProxyQuota, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.url, http.NoBody)
	if err != nil {
		return ProxyQuota{}, err
	}
	req.Header.Set("Authorization", "Basic "+d.apiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := d.hc.Do(req)
	if err != nil {
		return ProxyQuota{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ProxyQuota{}, fmt.Errorf("decodo subscriptions: status %d", resp.StatusCode)
	}
	var subs []decodoSubscription
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return ProxyQuota{}, fmt.Errorf("decodo subscriptions: %w", err)
	}
	if len(subs) == 0 {
		return ProxyQuota{}, errors.New("decodo subscriptions: none returned")
	}
	return subs[0].quota()
}

func (s decodoSubscription) quota() (ProxyQuota, error) {
	used, err := strconv.ParseFloat(s.TrafficPerPeriod, 64)
	if err != nil {
		return ProxyQuota{}, fmt.Errorf("decodo traffic_per_period %q: %w", s.TrafficPerPeriod, err)
	}
	limit, err := strconv.ParseFloat(s.TrafficLimit, 64)
	if err != nil {
		return ProxyQuota{}, fmt.Errorf("decodo traffic_limit %q: %w", s.TrafficLimit, err)
	}
	resets, err := parseValidUntil(s.ValidUntil)
	if err != nil {
		return ProxyQuota{}, fmt.Errorf("decodo valid_until %q: %w", s.ValidUntil, err)
	}
	return ProxyQuota{Used: used, Limit: limit, ResetsAt: resets}, nil
}

func parseValidUntil(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse(time.DateOnly, s)
}

// ProxyUsageView is the body of GET /usage/proxies.
type ProxyUsageView struct {
	Providers []dto.QuotaView `json:"providers"`
}

// ProxyUsage holds the latest proxy quota snapshot in memory.
type ProxyUsage struct {
	fetcher QuotaFetcher
	now     func() time.Time

	mu   sync.Mutex
	view dto.QuotaView
}

func newProxyUsage(fetcher QuotaFetcher, now func() time.Time) *ProxyUsage {
	if now == nil {
		now = time.Now
	}
	u := &ProxyUsage{fetcher: fetcher, now: now}
	u.view = dto.QuotaView{Provider: proxyProvider, Unit: proxyUnit, Level: dto.UsageOK}
	if fetcher == nil {
		u.view.Status = dto.QuotaNotConfigured
	}
	return u
}

// Refresh fetches the quota and replaces the snapshot. A failed fetch keeps
// the last good numbers and records the error; it is logged, not returned.
func (u *ProxyUsage) Refresh(ctx context.Context) error {
	if u.fetcher == nil {
		return nil
	}
	quota, err := u.fetcher.FetchQuota(ctx)
	now := u.now()

	u.mu.Lock()
	defer u.mu.Unlock()
	if err != nil {
		slog.WarnContext(ctx, "proxy quota fetch failed", slog.String(logger.KeyProvider, proxyProvider), slog.Any(logger.KeyErr, err))
		u.view.Status = dto.QuotaError
		u.view.Error = err.Error()
		return nil
	}
	view := dto.QuotaView{
		Provider:  proxyProvider,
		Status:    dto.QuotaOK,
		Used:      &quota.Used,
		Limit:     &quota.Limit,
		Unit:      proxyUnit,
		ResetsAt:  &quota.ResetsAt,
		FetchedAt: &now,
	}
	view.Percent, view.Level = dto.QuotaPercent(view.Used, view.Limit)
	u.view = view
	return nil
}

// View returns the current snapshot.
func (u *ProxyUsage) View(context.Context, string) (ProxyUsageView, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return ProxyUsageView{Providers: []dto.QuotaView{u.view}}, nil
}
