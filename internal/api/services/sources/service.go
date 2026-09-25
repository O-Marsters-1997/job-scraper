package sources

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) List(context.Context, string) ([]sourcespec.SourceInfo, error) {
	return sourcespec.Sources(), nil
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
