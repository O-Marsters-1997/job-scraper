package dto

import "time"

// RejectionKind says which gate turned a Job away.
type RejectionKind string

const (
	RejectionKindFilter RejectionKind = "filter"
	RejectionKindAge    RejectionKind = "age"
)

// JobRejection records why one user's search config turned one Job away.
// ConfigVersion is the config's UpdatedAt at rejection time, zero when the
// user had no saved config. IsCurrent is set only on reads.
type JobRejection struct {
	JobID         string
	UserID        string
	Kind          RejectionKind
	Reason        string
	ConfigVersion time.Time
	RejectedAt    time.Time
	IsCurrent     bool
}
