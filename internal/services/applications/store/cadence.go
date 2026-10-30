package store

import (
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	firstFollowUpDays = 5
	laterFollowUpDays = 7
	revisitDays       = 90
	maxSendsPerReply  = 3
)

func addDays(today time.Time, days int) *time.Time {
	y, m, d := today.UTC().Date()
	t := time.Date(y, m, d+days, 0, 0, 0, 0, time.UTC)
	return &t
}

func NextSend(sentBefore int, today time.Time) (followUpBy *time.Time, closes bool) {
	switch {
	case sentBefore+1 >= maxSendsPerReply:
		return nil, true
	case sentBefore == 0:
		return addDays(today, firstFollowUpDays), false
	default:
		return addDays(today, laterFollowUpDays), false
	}
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
