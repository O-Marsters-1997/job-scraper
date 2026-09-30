package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/ollymarsters/job-scraper/internal/queue"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: queue count | list [limit] | inspect <task-id> | replay <task-id>")
	}
	q, err := queue.NewBrokerFromEnv()
	if err != nil {
		fail(err.Error())
	}
	defer func() { _ = q.Close() }()
	ctx := context.Background()
	switch os.Args[1] {
	case "count":
		count, err := q.DeadLetterCount()
		if err != nil {
			fail(err.Error())
		}
		fmt.Println(count)
	case "list", "inspect":
		limit := 100
		if os.Args[1] == "list" && len(os.Args) > 2 {
			limit, err = strconv.Atoi(os.Args[2])
			if err != nil {
				fail(err.Error())
			}
		}
		if os.Args[1] == "inspect" && len(os.Args) != 3 {
			fail("usage: queue inspect <task-id>")
		}
		letters, err := q.DeadLetters(ctx, limit)
		if err != nil {
			fail(err.Error())
		}
		for _, letter := range letters {
			if os.Args[1] == "inspect" && letter.Task.ID != os.Args[2] {
				continue
			}
			encoded, err := json.Marshal(letter)
			if err != nil {
				fail(err.Error())
			}
			fmt.Println(string(encoded))
		}
	case "replay":
		if len(os.Args) != 3 {
			fail("usage: queue replay <task-id>")
		}
		if err := q.ReplayDead(ctx, os.Args[2]); err != nil {
			fail(err.Error())
		}
		fmt.Println("replayed", os.Args[2])
	default:
		fail("unknown command")
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
