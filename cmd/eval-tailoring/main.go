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
	suggest := flag.Bool("suggest", false, "evaluate inline suggestions instead of generation")
	flag.Parse()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fail("OPENROUTER_API_KEY is not set")
	}
	client := cvedit.NewClient()
	if *suggest {
		runSuggest(client, apiKey, *runs, *only)
		return
	}
	fixtures, err := eval.Fixtures()
	if err != nil {
		fail(err.Error())
	}

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
	fmt.Print(eval.Report(cvedit.PromptVersion, cvedit.Model, outcomes))
}

func runSuggest(client *cvedit.Client, apiKey string, runs int, only string) {
	fixtures, err := eval.SuggestFixtures()
	if err != nil {
		fail(err.Error())
	}
	var outcomes []eval.SuggestOutcome
	for _, f := range fixtures {
		if !strings.Contains(f.Name, only) {
			continue
		}
		for range runs {
			o, err := eval.RunSuggest(context.Background(), client, apiKey, f)
			if err != nil {
				fail(err.Error())
			}
			outcomes = append(outcomes, o)
		}
	}
	if len(outcomes) == 0 {
		fail("no fixture matched")
	}
	fmt.Print(eval.SuggestReport(cvedit.SuggestPromptVersion, cvedit.SuggestModel, outcomes))
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
