package db_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestUpsertSearchConfig_ScoringQuestionsRoundTrip(t *testing.T) {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "scoring-questions-user", "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	questions := dto.ScoringQuestions{
		Profile: "Senior Go engineer",
		Criteria: []dto.ScoringCriterion{
			{Key: "go_backend", Instructions: "Does the job use Go?", True: "yes", False: "no", Required: true},
		},
		Scale: []string{"Not relevant", "Weak", "Possible", "Strong", "Apply today"},
	}

	if _, err := testDB.UpsertSearchConfig(ctx, dto.SearchConfig{UserID: user.ID, ScoringQuestions: questions}); err != nil {
		t.Fatal(err)
	}

	got, err := testDB.GetSearchConfig(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.ScoringQuestions, questions) {
		t.Fatalf("scoring questions = %+v, want %+v", got.ScoringQuestions, questions)
	}
}
