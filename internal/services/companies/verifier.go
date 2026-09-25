package companies

import (
	"context"
	"errors"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/builder"
	"github.com/ollymarsters/job-scraper/internal/sources/registry"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type ATSBoardVerifier struct{}

func (ATSBoardVerifier) Verify(ctx context.Context, source, token string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if role, ok := sourcespec.SourceRole(source); !ok || role != sourcespec.RoleATS || !sourcespec.ValidBoardToken(token) {
		return errors.New("unsupported board configuration")
	}
	if source == "greenhouse" {
		entry, err := registry.Open(dto.SourceTarget{Source: source, Value: token, Enabled: true})
		if err != nil {
			return err
		}
		return verifyPages(ctx, entry.Source)
	}
	srcs := builder.BuildSources([]dto.SourceTarget{{Source: source, Value: token, Enabled: true}})
	if len(srcs) != 1 {
		return errors.New("unsupported board source")
	}
	return srcs[0].Iterate(ctx, func(context.Context, []dto.Job) (bool, error) { return false, nil })
}

func verifyPages(ctx context.Context, src sources.PageSource) error {
	for cursor := ""; ; {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := src.FetchPage(ctx, cursor)
		if err != nil {
			return err
		}
		if page.NextCursor == "" {
			return nil
		}
		if page.NextCursor == cursor {
			return errors.New("source returned unchanged cursor")
		}
		cursor = page.NextCursor
	}
}
