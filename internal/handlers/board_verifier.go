package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources/builder"
)

type ATSBoardVerifier struct{}

func (ATSBoardVerifier) Verify(ctx context.Context, source, token string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	srcs := builder.BuildSources([]dto.SourceTarget{{Source: source, Value: token, Enabled: true}}, nil)
	if len(srcs) != 1 {
		return errors.New("unsupported board source")
	}
	return srcs[0].Iterate(ctx, func(context.Context, []dto.Job) (bool, error) { return false, nil })
}
