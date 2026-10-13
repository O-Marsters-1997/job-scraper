package dto

import "time"

type ContactChannel string

const (
	ChannelLinkedInMessage ContactChannel = "linkedin_message"
	ChannelEmail           ContactChannel = "email"
	ChannelCall            ContactChannel = "call"
	ChannelInbound         ContactChannel = "inbound"
)

type ContactOutcome string

const (
	OutcomeNoReply   ContactOutcome = "no_reply"
	OutcomeRejected  ContactOutcome = "rejected"
	OutcomeSpoke     ContactOutcome = "spoke"
	OutcomeSubmitted ContactOutcome = "submitted"
)

type RecruiterContact struct {
	ID          string         `json:"id"`
	JobID       string         `json:"job_id"`
	ContactedOn string         `json:"contacted_on"`
	Channel     ContactChannel `json:"channel"`
	Outcome     ContactOutcome `json:"outcome"`
	CreatedAt   time.Time      `json:"created_at"`
}

type LogRecruiterContactInput struct {
	JobID       string         `json:"-" path:"id"`
	ContactedOn string         `json:"contacted_on"`
	Channel     ContactChannel `json:"channel"`
	Outcome     ContactOutcome `json:"outcome"`
}

// RecruiterContactSummariesQuery is answered as map[jobID]RecruiterContactSummary; jobs with no contact are absent.
type RecruiterContactSummariesQuery struct {
	JobIDs string `json:"job_ids"` // comma-separated
}

type RecruiterContactSummary struct {
	Count           int            `json:"count"`
	LastContactedOn string         `json:"last_contacted_on"`
	BestOutcome     ContactOutcome `json:"best_outcome"`
}

// Rank orders outcomes submitted > spoke > rejected > no_reply; unknown outcomes rank lowest.
func (o ContactOutcome) Rank() int {
	switch o {
	case OutcomeSubmitted:
		return 4
	case OutcomeSpoke:
		return 3
	case OutcomeRejected:
		return 2
	case OutcomeNoReply:
		return 1
	}
	return 0
}
