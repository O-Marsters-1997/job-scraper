package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type SourceTargetProvider interface {
	ListSourceTargetsByUser(ctx context.Context, userID string) ([]dto.SourceTarget, error)
	ListEnabledSourceTargets(ctx context.Context) ([]dto.SourceTarget, error)
	CreateSourceTarget(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error)
	UpdateSourceTarget(ctx context.Context, id, userID string, enabled bool) (dto.SourceTarget, error)
	DeleteSourceTarget(ctx context.Context, id, userID string) error
}
