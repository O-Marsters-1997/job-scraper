package scraper

import (
	"context"
	"errors"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
)

func VerifyBoard(ctx context.Context, source, token string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if role, ok := sourcespec.SourceRole(source); !ok || role != sourcespec.RoleATS || !sourcespec.ValidBoardToken(token) {
		return errors.New("unsupported board configuration")
	}
	src, ok := builder.BuildSource(dto.SourceTarget{Source: source, Value: token, Enabled: true})
	if !ok {
		return errors.New("unsupported board source")
	}
	_, _, err := src.FetchPage(ctx, "")
	return err
}
