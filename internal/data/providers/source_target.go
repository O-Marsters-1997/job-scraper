package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type SourceTargetProvider interface {
	ListSourceTargetsByUser(ctx context.Context, userID string) ([]dto.SourceTarget, error)
	ListEnabledSourceTargets(ctx context.Context) ([]dto.SourceTarget, error)
	CreateSourceTarget(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error)
	CreateSourceTargetWithRun(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error)
	UpsertSourceTargetForCompany(ctx context.Context, userID, source, value, companyID string, enabled bool, interval int) (dto.SourceTarget, error)
	UpdateSourceTarget(ctx context.Context, id, userID string, enabled *bool, checkIntervalMinutes *int) (dto.SourceTarget, error)
	DeleteSourceTarget(ctx context.Context, id, userID string) error
	SetSourceTargetRunState(ctx context.Context, id, status, runError string) (dto.SourceTarget, error)
	StartSourceTargetRun(ctx context.Context, id string) (dto.SourceTarget, error)
	GetVerifiedBoardID(ctx context.Context, source, token string) (string, error)
}
