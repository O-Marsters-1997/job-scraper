package cvtailor_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func ptr(s string) *string { return &s }

func TestCreatePositionValidation(t *testing.T) {
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil)
	cases := []struct {
		name string
		in   dto.PositionInput
	}{
		{"blank employer", dto.PositionInput{Employer: "  ", Title: "Engineer"}},
		{"blank title", dto.PositionInput{Employer: "Acme"}},
		{"bad date", dto.PositionInput{Employer: "Acme", Title: "Engineer", StartDate: ptr("last May")}},
		{"end before start", dto.PositionInput{Employer: "Acme", Title: "Engineer", StartDate: ptr("2022-01-01"), EndDate: ptr("2021-01-01")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreatePosition(context.Background(), "u1", tc.in)
			if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
				t.Fatalf("CreatePosition() err = %v, want an invalid error", err)
			}
		})
	}
}

func TestCreatePositionTreatsBlankDatesAsCurrent(t *testing.T) {
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil)
	got, err := svc.CreatePosition(context.Background(), "u1", dto.PositionInput{
		Employer: " Acme ", Title: "Engineer", StartDate: ptr("2020-01-01"), EndDate: ptr(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Employer != "Acme" || got.EndDate != nil {
		t.Fatalf("CreatePosition() = %+v, want trimmed employer and nil end date", got)
	}
}

func TestCreateAchievementRejectsBlankText(t *testing.T) {
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil)
	_, err := svc.CreateAchievement(context.Background(), "u1", dto.AchievementInput{PositionID: "p", Text: " "})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("CreateAchievement() err = %v, want an invalid error", err)
	}
}

func TestReorderRejectsRepeatedIDs(t *testing.T) {
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), nil, nil)
	_, err := svc.ReorderPositions(context.Background(), "u1", dto.ReorderInput{IDs: []string{"a", "a"}})
	if status, ok := apperr.StatusFor(err); !ok || status != apperr.KindInvalid.Status() {
		t.Fatalf("ReorderPositions() err = %v, want an invalid error", err)
	}
}
