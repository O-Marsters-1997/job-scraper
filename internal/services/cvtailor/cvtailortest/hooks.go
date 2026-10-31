package cvtailortest

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/outreachhook"
)

type RecordingHooks struct {
	Reply  outreachhook.Result
	Err    error
	Inputs []outreachhook.Input
}

func (h *RecordingHooks) Write(_ context.Context, _ string, in outreachhook.Input) (outreachhook.Result, error) {
	h.Inputs = append(h.Inputs, in)
	return h.Reply, h.Err
}

var _ cvtailor.HookWriter = (*RecordingHooks)(nil)
