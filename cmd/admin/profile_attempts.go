package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
)

const profileFixtureDir = "internal/services/cvtailor/eval/local/profile"

func runProfileAttempts(args []string) {
	if len(args) != 2 || args[0] != "export" {
		usage()
	}

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()

	userID, err := identity.NewFacade(pool, nil).UserIDByUsername(ctx, args[1])
	if err != nil {
		fatal("find user", err)
	}
	captures, err := cvtailor.New(pool, nil, nil, nil, nil).ProfileCaptures(ctx, userID)
	if err != nil {
		fatal("list profile captures", err)
	}
	n, err := writeProfileFixtures(profileFixtureDir, captures)
	if err != nil {
		fatal("write fixtures", err)
	}
	fmt.Printf("%d fixtures written\n", n)
}

func writeProfileFixtures(dir string, captures []dto.ProfileCapture) (int, error) {
	if len(captures) == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return 0, fmt.Errorf("create %s: %w", dir, err)
	}
	for i, c := range captures {
		f := eval.ProfileFixture{
			Name: c.DraftID, Profile: c.BaseProfile, Final: c.FinalProfile,
			BankSkills: c.BankSkills, Bank: c.Bank,
			JobTitle: c.JobTitle, Company: c.Company, JobDescription: c.JobDescription,
		}
		if len(c.Attempts) > 0 {
			f.Steer = c.Attempts[0].Steer
		}
		raw, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			return i, fmt.Errorf("encode %s: %w", c.DraftID, err)
		}
		if err := os.WriteFile(filepath.Join(dir, c.DraftID+".json"), raw, 0o600); err != nil {
			return i, fmt.Errorf("write %s: %w", c.DraftID, err)
		}
	}
	return len(captures), nil
}
