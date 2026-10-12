package scoring_test

import (
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
)

func Example_classifyOncePerCompany() {
	companies := &scoringtest.SuggestedKinds{}
	m := scoring.Build(scoring.Deps{Store: scoringtest.NewFakeStore(), Companies: companies})
	_ = m

	questions := []string{scoring.AgencyQuestion}
	_ = questions

	kind := scoring.KindFromAnswer(dto.Answer{PYes: 0.9, PNo: 0.1, Confidence: 0.8})
	_ = kind
}
