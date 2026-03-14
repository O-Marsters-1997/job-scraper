package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ollymarsters/job-scraper/internal/sources/greenhouse"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	scraper, err := greenhouse.New(greenhouse.Config{
		BoardTokens: []string{"greenhouse"},
	})
	if err != nil {
		log.Fatal(err)
	}

	jobs, err := scraper.FetchJobs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

	fmt.Printf("fetched %d jobs from %s\n\n", len(jobs), scraper.Name())
	for _, j := range jobs {
		fmt.Printf("[%d] %s — %s\n    %s\n", j.ID, j.Title, j.Location, j.URL)
	}
}
