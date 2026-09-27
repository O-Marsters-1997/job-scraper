package db_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestUpsertSearchConfig_PreferencesRoundTrip(t *testing.T) {
	ctx := context.Background()
	user, err := createTestUser(ctx, "preferences-user")
	if err != nil {
		t.Fatal(err)
	}
	prefs := dto.Preferences{
		Picks: []dto.Pick{
			{OptionID: "tech:go", Stance: "nice", Source: "manual"},
			{OptionID: "domain:gambling", Stance: "block", Source: "manual"},
		},
	}

	if _, err := testDB.UpsertSearchConfig(ctx, dto.SearchConfig{UserID: user.ID, Preferences: prefs}); err != nil {
		t.Fatal(err)
	}

	got, err := testDB.GetSearchConfig(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Preferences, prefs) {
		t.Fatalf("preferences = %+v, want %+v", got.Preferences, prefs)
	}
}
