package eventstest

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/events"
)

type snapshotter func() (*dto.JobScoreEvidence, error)

func (s snapshotter) ScoreSnapshot(context.Context, string, string) (*dto.JobScoreEvidence, error) {
	return s()
}

// ScoresAs snapshots every job as evidence.
func ScoresAs(evidence dto.JobScoreEvidence) events.ScoreSnapshotter {
	return snapshotter(func() (*dto.JobScoreEvidence, error) { return &evidence, nil })
}

// Unscored reports every job as having no score.
func Unscored() events.ScoreSnapshotter {
	return snapshotter(func() (*dto.JobScoreEvidence, error) { return nil, nil })
}

// Fails makes every snapshot return err.
func Fails(err error) events.ScoreSnapshotter {
	return snapshotter(func() (*dto.JobScoreEvidence, error) { return nil, err })
}
