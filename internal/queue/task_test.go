package queue

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestTaskValidate(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name  string
		task  Task
		valid bool
	}{
		{"detail", Task{Version: 1, ID: id, Source: "wis", Kind: DetailTask, URL: "https://workinstartups.com/job/1", Card: dto.Job{Source: "wis"}}, true},
		{"missing source", Task{Version: 1, ID: id, Kind: DetailTask, URL: "https://example.com"}, false},
		{"unknown source", Task{Version: 1, ID: id, Source: "other", Kind: DetailTask, URL: "https://example.com"}, false},
		{"listing", Task{Version: 1, ID: id, Source: "linkedin", Kind: ListingPageTask, TargetID: id, RunID: id}, true},
		{"listing without run", Task{Version: 1, ID: id, Source: "linkedin", Kind: ListingPageTask, TargetID: id}, false},
		{"board", Task{Version: 1, ID: id, Source: "greenhouse", Kind: BoardCheckTask, BoardID: id}, true},
		{"wrong board source", Task{Version: 1, ID: id, Source: "wis", Kind: BoardCheckTask, BoardID: id}, false},
		{"board verify", Task{Version: 1, ID: id, Source: "greenhouse", Kind: BoardVerifyTask, CompanyID: id, BoardToken: "acme"}, true},
		{"board verify without company", Task{Version: 1, ID: id, Source: "greenhouse", Kind: BoardVerifyTask, BoardToken: "acme"}, false},
		{"board verify with bad token", Task{Version: 1, ID: id, Source: "greenhouse", Kind: BoardVerifyTask, CompanyID: id, BoardToken: "a/b"}, false},
		{"board verify on discovery source", Task{Version: 1, ID: id, Source: "wis", Kind: BoardVerifyTask, CompanyID: id, BoardToken: "acme"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.Validate() == nil; got != tt.valid {
				t.Fatalf("Validate success = %v, want %v", got, tt.valid)
			}
		})
	}
}
