package linkedin

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/proxy"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type Scraper struct{ sources.PaginatedBase }

var _ sources.Source = (*Scraper)(nil)

func New() *Scraper {
	return &Scraper{sources.NewBase(sources.Config{
		Name:              "linkedin",
		URLPrefix:         "https://www.linkedin.com/jobs",
		Schedule:          "0 */6 * * *",
		MinScrapeInterval: 5 * time.Hour,
		ProxyTier:         proxy.Residential,
	})}
}

func (s *Scraper) NeedsDetail() bool { return true }

func (s *Scraper) CanHandle(url string) bool {
	return strings.Contains(url, "linkedin.com")
}

// Iterate logs a warning and returns immediately — LinkedIn blocks scraping.
func (s *Scraper) Iterate(_ context.Context, _ func(context.Context, []dto.Job) (bool, error)) error {
	slog.Warn("linkedin: scraping not yet implemented; skipping")
	return nil
}

func (s *Scraper) GetDetails(_ context.Context, _ string) (dto.Job, error) {
	return dto.Job{}, errors.New("linkedin: GetDetails not implemented")
}
