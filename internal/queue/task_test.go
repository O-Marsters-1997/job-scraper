package queue_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
)

func base(kind queue.TaskKind) queue.Task {
	switch kind {
	case queue.ListingPageTask:
		return queuetest.ListingTask("linkedin")
	case queue.BoardCheckTask:
		return queue.Task{Version: 1, ID: uuid.NewString(), Source: "greenhouse", Kind: kind, BoardID: uuid.NewString()}
	case queue.BoardVerifyTask:
		return queue.Task{Version: 1, ID: uuid.NewString(), Source: "greenhouse", Kind: kind, CompanyID: uuid.NewString(), BoardToken: "acme"}
	case queue.DetailTask:
		return queuetest.DetailTask("wis")
	}
	panic("unknown task kind " + string(kind))
}

func TestTaskValidate(t *testing.T) {
	t.Run("accepts", func(t *testing.T) {
		for _, kind := range []queue.TaskKind{queue.DetailTask, queue.ListingPageTask, queue.BoardCheckTask, queue.BoardVerifyTask} {
			t.Run(string(kind), func(t *testing.T) {
				if err := base(kind).Validate(); err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
			})
		}
	})

	rejects := []struct {
		name   string
		kind   queue.TaskKind
		mutate func(*queue.Task)
	}{
		{"unsupported version", queue.DetailTask, func(k *queue.Task) { k.Version = 2 }},
		{"non-UUID task ID", queue.DetailTask, func(k *queue.Task) { k.ID = "not-a-uuid" }},
		{"missing source", queue.DetailTask, func(k *queue.Task) { k.Source = "" }},
		{"unknown source", queue.DetailTask, func(k *queue.Task) { k.Source = "other" }},
		{"unknown kind", queue.DetailTask, func(k *queue.Task) { k.Kind = "bogus" }},
		{"non-UUID target", queue.ListingPageTask, func(k *queue.Task) { k.TargetID = "target" }},
		{"non-UUID run", queue.ListingPageTask, func(k *queue.Task) { k.RunID = "run" }},
		{"target without run", queue.ListingPageTask, func(k *queue.Task) { k.RunID = "" }},
		{"listing on ATS source", queue.ListingPageTask, func(k *queue.Task) { k.Source = "greenhouse" }},
		{"detail URL without host", queue.DetailTask, func(k *queue.Task) { k.URL = "https:///job" }},
		{"detail URL with credentials", queue.DetailTask, func(k *queue.Task) { k.URL = "https://user:pw@example.com/job" }},
		{"detail URL with other scheme", queue.DetailTask, func(k *queue.Task) { k.URL = "ftp://example.com/job" }},
		{"detail card source mismatch", queue.DetailTask, func(k *queue.Task) { k.Card.Source = "linkedin" }},
		{"board check on discovery source", queue.BoardCheckTask, func(k *queue.Task) { k.Source = "wis" }},
		{"board check without board ID", queue.BoardCheckTask, func(k *queue.Task) { k.BoardID = "" }},
		{"board verify without company", queue.BoardVerifyTask, func(k *queue.Task) { k.CompanyID = "" }},
		{"board verify with bad token", queue.BoardVerifyTask, func(k *queue.Task) { k.BoardToken = "a/b" }},
		{"board verify on discovery source", queue.BoardVerifyTask, func(k *queue.Task) { k.Source = "wis" }},
	}
	for _, tt := range rejects {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			task := base(tt.kind)
			tt.mutate(&task)
			if err := task.Validate(); err == nil {
				t.Fatalf("Validate(%+v) = nil, want error", task)
			}
		})
	}
}
