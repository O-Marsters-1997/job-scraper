package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

func main() {
	runs := flag.Int("runs", 1, "runs per fixture")
	only := flag.String("fixture", "", "run only fixtures whose name contains this")
	suite := flag.String("suite", "suggest", "suggest, profile or hook")
	local := flag.String("local", "internal/services/cvtailor/eval/local/profile", "directory of local profile fixtures")
	flag.Parse()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fail("OPENROUTER_API_KEY is not set")
	}
	model := lineedit.NewOpenRouter()
	var report string
	var err error
	switch *suite {
	case "suggest":
		report, err = runSuggest(apiKey, lineedit.New(model), *only, *runs)
	case "profile":
		report, err = runProfile(apiKey, profiledraft.New(model), *local, *only, *runs)
	case "hook":
		report, err = runHook(apiKey, model, *only, *runs)
	default:
		fail("unknown suite " + *suite)
	}
	if err != nil {
		fail(err.Error())
	}
	fmt.Print(report)
}

func runSuggest(apiKey string, editor *lineedit.Editor, only string, runs int) (string, error) {
	fixtures, err := eval.SuggestFixtures()
	if err != nil {
		return "", err
	}
	var outcomes []eval.SuggestOutcome
	for _, f := range fixtures {
		if !strings.Contains(f.Name, only) {
			continue
		}
		for range runs {
			o, err := eval.RunSuggest(context.Background(), editor, apiKey, f)
			if err != nil {
				return "", err
			}
			outcomes = append(outcomes, o)
		}
	}
	if len(outcomes) == 0 {
		return "", fmt.Errorf("no fixture matched")
	}
	return eval.SuggestReport(outcomes), nil
}

func runProfile(apiKey string, w *profiledraft.Writer, local, only string, runs int) (string, error) {
	fixtures, err := eval.ProfileFixtures(local)
	if err != nil {
		return "", err
	}
	var outcomes []eval.ProfileOutcome
	for _, f := range fixtures {
		if !strings.Contains(f.Name, only) {
			continue
		}
		for range runs {
			o, err := eval.RunProfile(context.Background(), w, apiKey, f)
			if err != nil {
				return "", err
			}
			outcomes = append(outcomes, o)
		}
	}
	if len(outcomes) == 0 {
		return "", fmt.Errorf("no fixture matched")
	}
	return eval.ProfileReport(outcomes), nil
}

func runHook(apiKey string, model lineedit.Model, only string, runs int) (string, error) {
	fixtures, err := eval.HookFixtures()
	if err != nil {
		return "", err
	}
	var outcomes []eval.HookOutcome
	for _, f := range fixtures {
		if !strings.Contains(f.Name, only) {
			continue
		}
		for range runs {
			o, err := eval.RunHook(context.Background(), model, apiKey, f)
			if err != nil {
				return "", err
			}
			outcomes = append(outcomes, o)
		}
	}
	if len(outcomes) == 0 {
		return "", fmt.Errorf("no fixture matched")
	}
	return eval.HookReport(outcomes), nil
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
