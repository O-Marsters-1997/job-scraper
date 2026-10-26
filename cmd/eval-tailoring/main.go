package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
)

func main() {
	runs := flag.Int("runs", 1, "runs per fixture")
	only := flag.String("fixture", "", "run only fixtures whose name contains this")
	flag.Parse()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fail("OPENROUTER_API_KEY is not set")
	}
	fixtures, err := eval.SuggestFixtures()
	if err != nil {
		fail(err.Error())
	}
	editor := lineedit.New(lineedit.NewOpenRouter())
	var outcomes []eval.SuggestOutcome
	for _, f := range fixtures {
		if !strings.Contains(f.Name, *only) {
			continue
		}
		for range *runs {
			o, err := eval.RunSuggest(context.Background(), editor, apiKey, f)
			if err != nil {
				fail(err.Error())
			}
			outcomes = append(outcomes, o)
		}
	}
	if len(outcomes) == 0 {
		fail("no fixture matched")
	}
	fmt.Print(eval.SuggestReport(outcomes))
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
