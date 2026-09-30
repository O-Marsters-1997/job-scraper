package queuetest

import (
	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

func DetailTask(source string) queue.Task {
	id := uuid.NewString()
	return queue.Task{Version: 1, ID: id, Source: source, Kind: queue.DetailTask, URL: "https://example.com/job/" + id, Card: dto.Job{Source: source}}
}

func ListingTask(source string) queue.Task {
	return queue.Task{Version: 1, ID: uuid.NewString(), Source: source, Kind: queue.ListingPageTask, TargetID: uuid.NewString(), RunID: uuid.NewString()}
}
