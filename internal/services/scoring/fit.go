package scoring

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

var errFitDisabled = apperr.Invalid("the weights fit is disabled until it is replaced (PRD B)")

func (s *Service) Fit(_ context.Context, _ string) (string, error) {
	return "", errFitDisabled
}
