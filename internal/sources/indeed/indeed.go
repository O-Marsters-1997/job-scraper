package indeed

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type Scraper struct{ sources.PaginatedBase }

var _ sources.Source = (*Scraper)(nil)
var _ sources.DetailFetcher = (*Scraper)(nil)

func New() *Scraper {
	return &Scraper{sources.NewBase(sources.Config{
		Name:              "indeed",
		URLPrefix:         "https://www.indeed.com",
		Schedule:          "0 */6 * * *",
		MinScrapeInterval: 5 * time.Hour,
		UseProxy:          true,
	})}
}

func (s *Scraper) CanHandle(url string) bool {
	return strings.Contains(url, "indeed.com")
}

// Iterate logs a warning and returns immediately — Indeed blocks scraping.
func (s *Scraper) Iterate(_ context.Context, _ func(context.Context, []dto.Job) (bool, error)) error {
	slog.Warn("indeed: scraping not yet implemented; skipping")
	return nil
}

func (s *Scraper) GetDetails(_ context.Context, _ string) (dto.Job, error) {
	return dto.Job{}, errors.New("indeed: GetDetails not implemented")
}
