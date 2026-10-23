package dto

import "time"

type ScrapeRunKind string

const ScrapeRunSearch ScrapeRunKind = "search"

type ScrapeRunTrigger string

const (
	TriggerScheduled ScrapeRunTrigger = "scheduled"
	TriggerManual    ScrapeRunTrigger = "manual"
)

type StopReason string

const (
	StopLastPage StopReason = "last_page"
	StopPageCap  StopReason = "page_cap"
	StopFailed   StopReason = "failed"
)

type BoardTally struct {
	Claimed                         bool
	Listed, New, Elsewhere, Removed int
}

type IngestTally struct {
	New, Merged int
}

type ScrapeRunOutcome string

const (
	OutcomeRunning   ScrapeRunOutcome = "running"
	OutcomeSucceeded ScrapeRunOutcome = "succeeded"
	OutcomeRetried   ScrapeRunOutcome = "retried"
	OutcomeFailed    ScrapeRunOutcome = "failed"
)

type ScrapeRun struct {
	ID            string
	Kind          ScrapeRunKind
	Source        string
	TargetID      string
	TargetValue   string
	TargetFilters map[string]string
	Trigger       ScrapeRunTrigger
	Outcome       ScrapeRunOutcome
	StopReason    StopReason
	StartedAt     time.Time
	FinishedAt    *time.Time

	Pages, Listings, New, Elsewhere, Repeat, Deferred, Retries int

	Errors []ScrapeRunError
}

type ScrapeRunError struct {
	Page    int
	Attempt int
	Message string
	At      time.Time
}

type ScrapeRunsQuery struct {
	Target string `json:"target"`
	Before string `json:"before"`
	Limit  string `json:"limit"`
}

type ScrapeRunPage struct {
	Items      []ScrapeRun
	NextCursor *string
}

type ScrapeRunListOptions struct {
	TargetID        string
	BeforeStartedAt time.Time
	BeforeID        string
	Limit           int
}
