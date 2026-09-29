package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
)

func main() {
	runs := flag.Int("runs", 1, "runs per fixture")
	only := flag.String("fixture", "", "run only fixtures whose name contains this")
	flag.Parse()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fail("OPENROUTER_API_KEY is not set")
	}
	fixtures, err := eval.Fixtures()
	if err != nil {
		fail(err.Error())
	}

	client := cvedit.NewClient()
	var outcomes []eval.Outcome
	for _, f := range fixtures {
		if !strings.Contains(f.Name, *only) {
			continue
		}
		for range *runs {
			o, err := eval.Run(context.Background(), client, apiKey, f)
			if err != nil {
				fail(err.Error())
			}
			outcomes = append(outcomes, o)
		}
	}
	if len(outcomes) == 0 {
		fail("no fixture matched")
	}
	eval.Report(os.Stdout, cvedit.PromptVersion, cvedit.Model, outcomes)
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
