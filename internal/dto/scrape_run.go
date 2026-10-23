<<<<<<< HEAD
package dto

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
||||||| parent of 5ab4f84f (Board polls report new, elsewhere and removed counts)
=======
package dto

type BoardTally struct {
	Claimed                         bool
	Listed, New, Elsewhere, Removed int
}

type IngestTally struct {
	New, Merged int
}
>>>>>>> 5ab4f84f (Board polls report new, elsewhere and removed counts)
