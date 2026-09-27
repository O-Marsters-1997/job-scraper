package telemetry

// EventTaskDone is the queue consumer's post-handler log event (ADR 0010).
const EventTaskDone = "task.done"

// EventHTTPRequest marks a completed API request (ADR 0010).
const EventHTTPRequest = "http.request"

// EventScoreCall marks one successful Jev answer call billed to a user.
const EventScoreCall = "score.call"
