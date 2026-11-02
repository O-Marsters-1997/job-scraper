package store

import (
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const revisitDays = 90

func addDays(today time.Time, days int) *time.Time {
	y, m, d := today.UTC().Date()
	t := time.Date(y, m, d+days, 0, 0, 0, 0, time.UTC)
	return &t
}

func ReplyState(outcome dto.ContactOutcome, today time.Time) (dto.ContactStage, *time.Time) {
	switch outcome {
	case dto.ContactOutcomeInterested, dto.ContactOutcomeReferred, dto.ContactOutcomeNoReply:
		return dto.ContactStageReplied, nil
	case dto.ContactOutcomeNotNow:
		return dto.ContactStageReplied, addDays(today, revisitDays)
	case dto.ContactOutcomeNoThanks:
		return dto.ContactStageClosed, nil
	}
	return dto.ContactStageReplied, nil
}
