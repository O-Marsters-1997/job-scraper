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

func (o ContactOutcome) Rank() int {
	panic("not implemented")
}

type RecruiterContact struct {
	ID          string         `json:"id"`
	JobID       string         `json:"job_id"`
	ContactedOn string         `json:"contacted_on"`
	Channel     ContactChannel `json:"channel"`
	Outcome     ContactOutcome `json:"outcome"`
	CreatedAt   time.Time      `json:"created_at"`
}

type LogRecruiterContactInput struct {
	JobID       string         `json:"job_id"`
	ContactedOn string         `json:"contacted_on"`
	Channel     ContactChannel `json:"channel"`
	Outcome     ContactOutcome `json:"outcome"`
}

type RecruiterContactsQuery struct {
	JobID string `json:"job_id"`
}

type RecruiterContactsForJobsQuery struct {
	JobIDs string `json:"job_ids"`
}

type RecruiterContactSummary struct {
	Count           int            `json:"count"`
	LastContactedOn string         `json:"last_contacted_on"`
	BestOutcome     ContactOutcome `json:"best_outcome"`
}
