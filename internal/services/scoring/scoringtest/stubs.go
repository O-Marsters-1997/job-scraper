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

var _ scoring.Reconsiderer = ReconsidererStub{}

type SuggestedKinds struct {
	Kinds map[string]dto.CompanyKind
}

func (s *SuggestedKinds) SuggestCompanyKind(_ context.Context, companyID string, kind dto.CompanyKind) (bool, error) {
	panic("not implemented")
}

var _ scoring.CompanyClassifier = (*SuggestedKinds)(nil)
