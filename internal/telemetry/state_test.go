package telemetry_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

type fakeStateReader struct {
	state dto.OpsState
	err   error
}

func (f fakeStateReader) OpsState(context.Context) (dto.OpsState, error) {
	return f.state, f.err
}

func TestStateCollector(t *testing.T) {
	tests := []struct {
		name   string
		reader fakeStateReader
		want   string
	}{
		{
			name: "happy path",
			reader: fakeStateReader{state: dto.OpsState{
				OutboxPending:          3,
				OutboxOldestPendingAge: 90 * time.Second,
				OutboxFailed:           1,
			}},
			want: `
				# HELP jobscraper_outbox_failed Scoring effects with status failed.
				# TYPE jobscraper_outbox_failed gauge
				jobscraper_outbox_failed 1
				# HELP jobscraper_outbox_oldest_pending_seconds Age in seconds of the oldest pending or running scoring effect.
				# TYPE jobscraper_outbox_oldest_pending_seconds gauge
				jobscraper_outbox_oldest_pending_seconds 90
				# HELP jobscraper_outbox_pending Scoring effects with status pending or running.
				# TYPE jobscraper_outbox_pending gauge
				jobscraper_outbox_pending 3
				# HELP jobscraper_state_up 1 if the last OpsState read succeeded, 0 otherwise.
				# TYPE jobscraper_state_up gauge
				jobscraper_state_up 1
			`,
		},
		{
			name:   "read failure emits only state_up 0",
			reader: fakeStateReader{err: errors.New("read failed")},
			want: `
				# HELP jobscraper_state_up 1 if the last OpsState read succeeded, 0 otherwise.
				# TYPE jobscraper_state_up gauge
				jobscraper_state_up 0
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := telemetry.NewStateCollector(tt.reader)
			if err := testutil.CollectAndCompare(collector, strings.NewReader(tt.want)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
