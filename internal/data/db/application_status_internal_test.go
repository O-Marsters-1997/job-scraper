package db

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/fp"
)

func TestToOptionalDateRejectsInvalidDate(t *testing.T) {
	_, err := toOptionalDate(fp.Some("not-a-date"))
	if err == nil {
		t.Fatal("expected an invalid date error")
	}
}
