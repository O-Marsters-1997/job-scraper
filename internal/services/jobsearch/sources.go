package jobsearch

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

func (s *Service) ListSources(context.Context, string) ([]sourcespec.SourceInfo, error) {
	return sourcespec.Sources(), nil
}

func (s *Service) ResolveBoard(_ context.Context, _ string, q dto.ResolveBoardQuery) (dto.ResolvedURL, error) {
	if q.URL == "" {
		return dto.ResolvedURL{}, apperr.Invalid("missing url")
	}
	if search, ok := detect.ParseSearchURL(q.URL); ok {
		return dto.ResolvedURL{
			Kind:    "search",
			Source:  search.Source,
			Value:   search.Value,
			Filters: search.Filters,
			Dropped: search.Dropped,
			URL:     detect.BuildSearchURL(search.Source, search.Value, search.Filters),
		}, nil
	}
	if source, token, ok := detect.ResolveBoard(q.URL); ok {
		return dto.ResolvedURL{
			Kind:    "ats",
			Source:  source,
			Value:   token,
			Filters: map[string]string{},
			Dropped: []string{},
			URL:     q.URL,
		}, nil
	}
	if label, ok := detect.UnsupportedBoard(q.URL); ok {
		return dto.ResolvedURL{}, apperr.Unprocessable(label + " searches aren't supported yet — use Build from fields")
	}
	return dto.ResolvedURL{}, apperr.Unprocessable("could not recognise a supported board or search page in that URL")
}
