// Package sources adapts the source registry and board-URL detection to the
// handler shapes: List and Resolve take (ctx, userID, ...) even though
// neither depends on the caller, so they can bind through the generic
// adapter instead of a closure in the router.
package sources

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources/registry"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) List(context.Context, string) ([]registry.SourceInfo, error) {
	return registry.Sources(), nil
}

func (s *Service) Resolve(_ context.Context, _ string, q dto.ResolveBoardQuery) (dto.ResolvedBoard, error) {
	if q.URL == "" {
		return dto.ResolvedBoard{}, apperr.Invalid("missing url")
	}
	source, token, ok := detect.ResolveBoard(q.URL)
	if !ok {
		return dto.ResolvedBoard{}, apperr.Unprocessable("could not resolve an ATS board from that URL")
	}
	return dto.ResolvedBoard{Source: source, Value: token}, nil
}
