package dto

import "time"

type (
	ContactStage     string
	ContactOutcome   string
	ContactChannel   string
	ContactEventKind string
	OutreachKind     string
)

const (
	ContactStageToContact ContactStage = "to_contact"
	ContactStageMessaged  ContactStage = "messaged"
	ContactStageReplied   ContactStage = "replied"
	ContactStageClosed    ContactStage = "closed"
)

const (
	ContactOutcomeInterested ContactOutcome = "interested"
	ContactOutcomeReferred   ContactOutcome = "referred"
	ContactOutcomeNotNow     ContactOutcome = "not_now"
	ContactOutcomeNoThanks   ContactOutcome = "no_thanks"
	ContactOutcomeNoReply    ContactOutcome = "no_reply"
)

const (
	ContactChannelLinkedIn ContactChannel = "linkedin"
	ContactChannelEmail    ContactChannel = "email"
	ContactChannelCall     ContactChannel = "call"
	ContactChannelInbound  ContactChannel = "inbound"
)

const (
	ContactEventAdded  ContactEventKind = "added"
	ContactEventSent   ContactEventKind = "sent"
	ContactEventReply  ContactEventKind = "reply"
	ContactEventClosed ContactEventKind = "closed"
)

const (
	OutreachRecruiter     OutreachKind = "recruiter"
	OutreachHiringManager OutreachKind = "hiring_manager"
	OutreachReferral      OutreachKind = "referral"
	OutreachFollowUp      OutreachKind = "follow_up"
)

type ContactEvent struct {
	ID           string           `json:"id"`
	Kind         ContactEventKind `json:"kind"`
	OccurredOn   string           `json:"occurred_on"`
	Channel      ContactChannel   `json:"channel,omitempty"`
	TemplateKind OutreachKind     `json:"template_kind,omitempty"`
	TemplateName string           `json:"template_name,omitempty"`
	Outcome      ContactOutcome   `json:"outcome,omitempty"`
	Note         string           `json:"note,omitempty"`
	JobID        *string          `json:"job_id"`
	JobTitle     string           `json:"job_title,omitempty"`
}

type Contact struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Role           string         `json:"role"`
	CompanyID      *string        `json:"company_id"`
	CompanyName    string         `json:"company_name"` // the Company's name when linked, else the typed fallback
	CompanyKind    string         `json:"company_kind"` // companies.kind; "" when unlinked
	JobID          *string        `json:"job_id"`
	JobTitle       string         `json:"job_title"`
	ApplicationID  *string        `json:"application_id"`
	LinkedInURL    string         `json:"linkedin_url"`
	Email          string         `json:"email"`
	Stage          ContactStage   `json:"stage"`
	Outcome        ContactOutcome `json:"outcome,omitempty"`
	FollowUpBy     *string        `json:"follow_up_by"`
	Reply          string         `json:"reply"`
	SentSinceReply int            `json:"sent_since_reply"`
	History        []ContactEvent `json:"history"`
	CreatedAt      time.Time      `json:"created_at"`
}

type ContactInput struct {
	ID          string  `json:"-" path:"id"`
	Name        string  `json:"name"`
	Role        string  `json:"role"`
	CompanyID   *string `json:"company_id"`
	CompanyName string  `json:"company_name"`
	JobID       *string `json:"job_id"`
	LinkedInURL string  `json:"linkedin_url"`
	Email       string  `json:"email"`
}

type ContactQuery struct {
	ID string `json:"-" path:"id"`
}

type MarkSentInput struct {
	ID           string         `json:"-" path:"id"`
	Channel      ContactChannel `json:"channel"`
	TemplateKind OutreachKind   `json:"template_kind"`
	TemplateName string         `json:"template_name"`
}

type ContactSent struct {
	Contact               Contact `json:"contact"`
	MovedApplicationChase bool    `json:"moved_application_chase"`
}

type RecordReplyInput struct {
	ID      string         `json:"-" path:"id"`
	Outcome ContactOutcome `json:"outcome"` // never no_reply
	Reply   string         `json:"reply"`
}
