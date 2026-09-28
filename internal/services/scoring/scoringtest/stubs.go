package scoringtest

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
)

// RecomputerStub satisfies scoringconfig.Recomputer with a primed result or
// error, in place of a real scoring.Service.
type RecomputerStub struct {
	result dto.RecomputeResult
	err    error
}

func Recomputes(n int64) RecomputerStub {
	return RecomputerStub{result: dto.RecomputeResult{Recomputed: n}}
}
func RecomputeFails(err error) RecomputerStub { return RecomputerStub{err: err} }
func (s RecomputerStub) Recompute(context.Context, string) (dto.RecomputeResult, error) {
	return s.result, s.err
}

// BackfillerStub satisfies scoringconfig.Backfiller with a primed result or
// error, in place of a real scoring.Service.
type BackfillerStub struct {
	queued int64
	err    error
}

func Backfills(n int64) BackfillerStub       { return BackfillerStub{queued: n} }
func BackfillFails(err error) BackfillerStub { return BackfillerStub{err: err} }
func (s BackfillerStub) FillMissingAnswers(context.Context, string) (int64, error) {
	return s.queued, s.err
}

// ReconsidererStub satisfies scoringconfig.Reconsiderer with a primed error,
// in place of jobsearch's real candidate reconsideration.
type ReconsidererStub struct{ err error }

func Reconsiders() ReconsidererStub              { return ReconsidererStub{} }
func ReconsiderFails(err error) ReconsidererStub { return ReconsidererStub{err: err} }
func (s ReconsidererStub) Reconsider(context.Context, dto.SearchConfig) error {
	return s.err
}

// ExtractorStub satisfies scoringconfig.Extractor with primed picks or an
// error, in place of a real LLM call.
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
	_ scoringconfig.Recomputer   = RecomputerStub{}
	_ scoringconfig.Backfiller   = BackfillerStub{}
	_ scoringconfig.Reconsiderer = ReconsidererStub{}
	_ scoringconfig.Extractor    = ExtractorStub{}
)
