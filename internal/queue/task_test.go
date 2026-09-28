package queue_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

func TestTaskValidate(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name  string
		task  queue.Task
		valid bool
	}{
		{"detail", queue.Task{Version: 1, ID: id, Source: "wis", Kind: queue.DetailTask, URL: "https://workinstartups.com/job/1", Card: dto.Job{Source: "wis"}}, true},
		{"missing source", queue.Task{Version: 1, ID: id, Kind: queue.DetailTask, URL: "https://example.com"}, false},
		{"unknown source", queue.Task{Version: 1, ID: id, Source: "other", Kind: queue.DetailTask, URL: "https://example.com"}, false},
		{"listing", queue.Task{Version: 1, ID: id, Source: "linkedin", Kind: queue.ListingPageTask, TargetID: id, RunID: id}, true},
		{"listing without run", queue.Task{Version: 1, ID: id, Source: "linkedin", Kind: queue.ListingPageTask, TargetID: id}, false},
		{"board", queue.Task{Version: 1, ID: id, Source: "greenhouse", Kind: queue.BoardCheckTask, BoardID: id}, true},
		{"wrong board source", queue.Task{Version: 1, ID: id, Source: "wis", Kind: queue.BoardCheckTask, BoardID: id}, false},
		{"board verify", queue.Task{Version: 1, ID: id, Source: "greenhouse", Kind: queue.BoardVerifyTask, CompanyID: id, BoardToken: "acme"}, true},
		{"board verify without company", queue.Task{Version: 1, ID: id, Source: "greenhouse", Kind: queue.BoardVerifyTask, BoardToken: "acme"}, false},
		{"board verify with bad token", queue.Task{Version: 1, ID: id, Source: "greenhouse", Kind: queue.BoardVerifyTask, CompanyID: id, BoardToken: "a/b"}, false},
		{"board verify on discovery source", queue.Task{Version: 1, ID: id, Source: "wis", Kind: queue.BoardVerifyTask, CompanyID: id, BoardToken: "acme"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.Validate() == nil; got != tt.valid {
				t.Fatalf("Validate success = %v, want %v", got, tt.valid)
			}
		})
	}
}
