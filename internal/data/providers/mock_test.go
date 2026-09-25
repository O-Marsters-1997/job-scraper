package providers

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestMockJobProviderPageAvailability(t *testing.T) {
	m := NewMockJobProvider()
	_, err := m.Save(context.Background(), []dto.Job{{URL: "https://example.com/job"}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := m.Page(context.Background(), "user", JobPageOptions{Limit: 10, Availability: "closed"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("closed jobs = %d, want 0", len(page.Items))
	}
}
