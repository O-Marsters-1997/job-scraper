struct ReplayOutcome {
    var result: ReplayResult
    var remaining: [QueuedAction]
    var stoppedBy: (any Error)?
}
