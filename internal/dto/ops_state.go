package dto

import "time"

type OpsState struct {
	OutboxPending          int64
	OutboxOldestPendingAge time.Duration
	OutboxFailed           int64
}
