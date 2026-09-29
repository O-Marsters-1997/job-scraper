package dto

import "time"

type OpsState struct {
	OutboxPending          int64
	OutboxOldestPendingAge time.Duration
	OutboxFailed           int64
	BoardsOverdue          int64
	BoardsFailing          int64
	SourceTargetsFailed    int64
	HarvestAge             map[string]time.Duration
}
