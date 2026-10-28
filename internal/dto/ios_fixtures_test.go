package dto_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var update = flag.Bool("update", false, "rewrite the iOS fixtures under ios/FastTrackTests/Fixtures")

func TestIOSFixtures(t *testing.T) {
	t.Parallel()
	score := 92
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	chaseBy := "2026-10-01"
	applyTaskID := "task-1"
	fixtures := map[string]any{
		"due_list": dto.DueList{Jobs: []dto.DueJob{
			{
				JobID: "job-1", Title: "Senior Go Engineer", CompanySlug: "monzo", CompanyKind: dto.KindAgency, Band: "great", Score: &score,
				ApplicationID: "app-1", ChaseBy: &chaseBy, ApplyTaskID: &applyTaskID,
				Tasks: []dto.DueTask{
					{ID: "task-1", TypeName: "Apply", SystemKey: dto.TaskTypeApply, Notes: "Tailor the CV"},
					{ID: "task-2", TypeName: "Finish application", SystemKey: dto.TaskTypeFinish, Notes: "Needs a cover letter"},
				},
				Draft:       &dto.DueDraft{ID: "draft-1", State: dto.DraftStateReady, CVTitle: "Backend"},
				KeptDraftID: new("draft-0"),
			},
			{JobID: "job-2", Title: "Backend Engineer", CompanySlug: "wise", CompanyKind: dto.KindUnclassified, ApplicationID: "app-2", Tasks: []dto.DueTask{}},
		}},
		"task_types": []dto.TaskType{
			{ID: "type-1", Name: "Finish application", Colour: "#0f766e", SystemKey: dto.TaskTypeFinish, CreatedAt: at},
			{ID: "type-2", Name: "Reach out", Colour: "#15803d", CreatedAt: at},
		},
		"recruiter_contact": dto.RecruiterContact{
			ID: "contact-1", JobID: "job-1", ContactedOn: "2026-10-01",
			Channel: dto.ChannelEmail, Outcome: dto.OutcomeNoReply, CreatedAt: at,
		},
		"job_page": dto.JobPage{
			Total: new(1),
			Items: []dto.Job{{
				ID:               "job-1",
				Title:            "Senior Go Engineer",
				Location:         "London",
				URL:              "https://example.com/jobs/1",
				CompanySlug:      "monzo",
				ScrapedAt:        at,
				SuitabilityScore: &score,
				Band:             "Great",
				CompanyKind:      dto.KindAgency,
				Breakdown: []dto.ScoreRow{
					{Key: "tech:go", Label: "Go", Stance: "want", Resolved: "yes", Effect: "meets"},
					{Key: "domain:fintech", Label: "Fintech", Stance: "want", Resolved: "no", Effect: "misses"},
				},
			}},
		},
	}

	for name, v := range fixtures {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join("..", "..", "ios", "FastTrackTests", "Fixtures", name+".json")
			got, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				t.Fatalf("marshal %s: %v", name, err)
			}
			got = append(got, '\n')

			if *update {
				if err := os.WriteFile(path, got, 0o600); err != nil {
					t.Fatalf("write %s: %v", path, err)
				}
				return
			}

			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v (run go test ./internal/dto -run TestIOSFixtures -update)", path, err)
			}
			var want, gotDecoded any
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatalf("decode %s: %v", path, err)
			}
			if err := json.Unmarshal(got, &gotDecoded); err != nil {
				t.Fatalf("decode marshalled %s: %v", name, err)
			}
			if diff := cmp.Diff(want, gotDecoded); diff != "" {
				t.Errorf("%s is stale; rerun with -update (-fixture +dto):\n%s", path, diff)
			}
		})
	}
}
