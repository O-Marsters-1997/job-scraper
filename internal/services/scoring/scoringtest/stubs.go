package scoringtest

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

type ReconsidererStub struct{ err error }

func Reconsiders() ReconsidererStub              { return ReconsidererStub{} }
func ReconsiderFails(err error) ReconsidererStub { return ReconsidererStub{err: err} }
func (s ReconsidererStub) Reconsider(context.Context, dto.SearchConfig) error {
	return s.err
}

type ExtractorStub struct {
	picks []dto.Pick
	err   error
}

func Extracts(picks []dto.Pick) ExtractorStub { return ExtractorStub{picks: picks} }
func ExtractFails(err error) ExtractorStub    { return ExtractorStub{err: err} }
func (s ExtractorStub) Extract(context.Context, string, string, []dto.ScoringOption, []dto.DimensionSpec) ([]dto.Pick, error) {
	return s.picks, s.err
}

var (
	_ scoring.Reconsiderer = ReconsidererStub{}
	_ scoring.Extractor    = ExtractorStub{}
)
