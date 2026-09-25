package applicationstatuses_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/api/services/applicationstatuses"
	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func assertKind(t *testing.T, err error, want apperr.Kind) {
	t.Helper()
	status, ok := apperr.StatusFor(err)
	if !ok || status != want.Status() {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, want.Status())
	}
}

func TestCreateRequiresNameAndColour(t *testing.T) {
	svc := applicationstatuses.New(&providers.MockApplicationStatusProvider{})
	_, err := svc.Create(context.Background(), "user-1", dto.ApplicationStatusInput{Name: "Offer"})
	assertKind(t, err, apperr.KindInvalid)
}

func TestCreateSucceeds(t *testing.T) {
	svc := applicationstatuses.New(&providers.MockApplicationStatusProvider{})
	got, err := svc.Create(context.Background(), "user-1", dto.ApplicationStatusInput{Name: "Offer", Colour: "#00ff00"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Offer" {
		t.Fatalf("name = %q, want Offer", got.Name)
	}
}

func TestDeleteRefusesAStatusInUse(t *testing.T) {
	store := &providers.MockApplicationStatusProvider{CountsInUse: map[string]int64{"s1": 3}}
	svc := applicationstatuses.New(store)

	err := svc.Delete(context.Background(), "user-1", "s1")

	assertKind(t, err, apperr.KindConflict)
	fields := apperr.FieldsFor(err)
	if fields["count"] != int64(3) {
		t.Fatalf("fields = %+v, want count=3", fields)
	}
}

func TestDeleteSucceedsWhenUnused(t *testing.T) {
	svc := applicationstatuses.New(&providers.MockApplicationStatusProvider{})
	if err := svc.Delete(context.Background(), "user-1", "s1"); err != nil {
		t.Fatal(err)
	}
}
