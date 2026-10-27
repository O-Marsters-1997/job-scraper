package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
)

func TestWriteProfileFixtures(t *testing.T) {
	t.Parallel()

	t.Run("each capture loads as a fixture with Final set", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "profile")
		captures := []dto.ProfileCapture{{
			DraftID: "d1", JobTitle: "Engineer", Company: "Acme", JobDescription: "Build", BaseProfile: "base", FinalProfile: "final",
			BankSkills: []string{"Go"}, Bank: []string{"Shipped x"}, Attempts: []dto.ProfileAttempt{{Steer: "calmer"}},
		}}

		n, err := writeProfileFixtures(dir, captures)
		if err != nil || n != 1 {
			t.Fatalf("writeProfileFixtures() = %d, %v, want 1", n, err)
		}

		fixtures, err := eval.ProfileFixtures(dir)
		if err != nil {
			t.Fatal(err)
		}
		var got *eval.ProfileFixture
		for i := range fixtures {
			if fixtures[i].Name == "d1" {
				got = &fixtures[i]
			}
		}
		if got == nil || got.Profile != "base" || got.Final != "final" || got.JobTitle != "Engineer" || got.Steer != "calmer" {
			t.Errorf("loaded fixture = %+v, want the capture's inputs with Final set", got)
		}
	})

	t.Run("no captures writes no files", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "profile")

		n, err := writeProfileFixtures(dir, nil)

		if _, statErr := os.Stat(dir); n != 0 || err != nil || !os.IsNotExist(statErr) {
			t.Errorf("writeProfileFixtures(nil) = %d, %v, stat %v, want 0 and no directory", n, err, statErr)
		}
	})
}
