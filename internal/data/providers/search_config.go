package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type SearchConfigProvider interface {
	GetSearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error)
	UpsertSearchConfig(ctx context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error)
}
