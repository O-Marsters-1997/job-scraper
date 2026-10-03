package dto

import "time"

type OpsState struct {
	OutboxPending          int64
	OutboxOldestPendingAge time.Duration
	OutboxFailed           int64
	BoardsOverdue          int64
	BoardsFailing          int64
	SourceTargetsFailed    int64
	DisabledSourceTargets  map[string]int64
	HarvestAge             map[string]time.Duration
	UniqueRelevantJobs     map[string]int64
	DiscoveryBoards        map[string]int64
	DiscoveryRelevantJobs  map[string]int64
	HarvestAdmitted        map[string]int64
	EmptiedBoards          map[string]int64
	UnderparsedBoards      map[string]int64
	FieldCompleteness      map[string]map[string]float64
}
