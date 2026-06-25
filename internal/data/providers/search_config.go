package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type SearchConfigProvider interface {
	ListSearchConfigs(ctx context.Context) ([]dto.SearchConfig, error)
	GetSearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error)
	UpsertSearchConfig(ctx context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error)
}
