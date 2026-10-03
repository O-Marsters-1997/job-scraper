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
				BoardsOverdue:          4,
				BoardsFailing:          2,
				SourceTargetsFailed:    5,
				DisabledSourceTargets:  map[string]int64{"indeed": 2},
				UniqueRelevantJobs:     map[string]int64{"lever": 7},
				HarvestAge:             map[string]time.Duration{"ashby": time.Hour, "lever": 30 * time.Second},
			}},
			want: `
				# HELP jobscraper_boards_failing Verified Boards with at least 3 consecutive failed polls.
				# TYPE jobscraper_boards_failing gauge
				jobscraper_boards_failing 2
				# HELP jobscraper_boards_overdue Verified, unleased Boards whose next_due_at has passed.
				# TYPE jobscraper_boards_overdue gauge
				jobscraper_boards_overdue 4
				# HELP jobscraper_harvest_age_seconds Seconds since each catalog harvester last succeeded.
				# TYPE jobscraper_harvest_age_seconds gauge
				jobscraper_harvest_age_seconds{harvester="ashby"} 3600
				jobscraper_harvest_age_seconds{harvester="lever"} 30
				# HELP jobscraper_outbox_failed Scoring effects with status failed.
				# TYPE jobscraper_outbox_failed gauge
				jobscraper_outbox_failed 1
				# HELP jobscraper_outbox_oldest_pending_seconds Age in seconds of the oldest pending or running scoring effect.
				# TYPE jobscraper_outbox_oldest_pending_seconds gauge
				jobscraper_outbox_oldest_pending_seconds 90
				# HELP jobscraper_outbox_pending Scoring effects with status pending or running.
				# TYPE jobscraper_outbox_pending gauge
				jobscraper_outbox_pending 3
				# HELP jobscraper_source_targets_disabled Source Targets that were disabled with a reason, per source.
				# TYPE jobscraper_source_targets_disabled gauge
				jobscraper_source_targets_disabled{source="indeed"} 2
				# HELP jobscraper_source_targets_failed Source Targets whose last run failed.
				# TYPE jobscraper_source_targets_failed gauge
				jobscraper_source_targets_failed 5
				# HELP jobscraper_source_unique_relevant_jobs Scored Jobs first discovered in the last 14 days whose every URL came from this source.
				# TYPE jobscraper_source_unique_relevant_jobs gauge
				jobscraper_source_unique_relevant_jobs{source="lever"} 7
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
				t.Errorf("CollectAndCompare() mismatch: %v", err)
			}
		})
	}
}
