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
	New, Merged, Upgraded int
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
	Q       string `json:"q"`
	Outcome string `json:"outcome"`
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Trigger string `json:"trigger"`
	Target  string `json:"target"`
	Range   string `json:"range"`
	From    string `json:"from"`
	To      string `json:"to"`
	Before  string `json:"before"`
	Limit   string `json:"limit"`
}

type ScrapeRunFilter struct {
	TargetID string
	Q        string
	Outcome  []string
	Kind     []string
	Source   []string
	Trigger  []string
	From, To time.Time
}

type ScrapeRunSummaryOptions struct {
	ScrapeRunFilter
	Bucket time.Duration
	End    time.Time
}

type ScrapeRunFacet string

const (
	FacetOutcome ScrapeRunFacet = "outcome"
	FacetKind    ScrapeRunFacet = "kind"
	FacetSource  ScrapeRunFacet = "source"
	FacetTrigger ScrapeRunFacet = "trigger"
)

type ScrapeRunSummary struct {
	Total     int
	Facets    map[ScrapeRunFacet]map[string]int
	BucketMs  int64
	Histogram []HistogramBucket
}

type HistogramBucket struct {
	Start               time.Time
	OK, Retried, Failed int
}

type ScrapeRunPage struct {
	Items      []ScrapeRun
	NextCursor *string
}

type ScrapeRunListOptions struct {
	ScrapeRunFilter
	BeforeStartedAt time.Time
	BeforeID        string
	Limit           int
}

type HealthStatus string

const (
	HealthRed   HealthStatus = "red"
	HealthAmber HealthStatus = "amber"
	HealthGreen HealthStatus = "green"
)

type HealthReason string

const (
	ReasonLastFailed    HealthReason = "last_failed"
	ReasonFailingStreak HealthReason = "failing_streak"
	ReasonRetried       HealthReason = "retried"
	ReasonNoNew         HealthReason = "no_new"
	ReasonNoRuns        HealthReason = "no_runs"
	ReasonOK            HealthReason = "ok"
)

type SourceHealth struct {
	Kind          ScrapeRunKind
	TargetID      string
	Source        string
	TargetValue   string
	TargetFilters map[string]string
	Status        HealthStatus
	Reason        HealthReason
	Recent        []ScrapeRunOutcome
	LastRunAt     *time.Time

	New, Elsewhere, Retries int
}
