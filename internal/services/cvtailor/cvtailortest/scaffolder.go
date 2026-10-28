package cvtailortest

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/letterwrite"
)

type RecordingScaffolder struct {
	replies []letterwrite.Result
	Err     error
	Inputs  []letterwrite.Input
}

func Scaffolding(replies ...letterwrite.Result) *RecordingScaffolder {
	return &RecordingScaffolder{replies: replies}
}

func (s *RecordingScaffolder) Scaffold(_ context.Context, _ string, in letterwrite.Input) (letterwrite.Result, error) {
	s.Inputs = append(s.Inputs, in)
	if s.Err != nil {
		return letterwrite.Result{}, s.Err
	}
	i := min(len(s.Inputs), len(s.replies)) - 1
	return s.replies[i], nil
}

var _ cvtailor.Scaffolder = (*RecordingScaffolder)(nil)
