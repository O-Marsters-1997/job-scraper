package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ollymarsters/job-scraper/internal/queue"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: queue <stats|dead|replay> <detail|scrape-request> [id]")
		os.Exit(2)
	}
	kind := queue.Kind(os.Args[2])
	addr := os.Getenv("VALKEY_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	q, err := queue.New(addr)
	if err != nil {
		fail(err)
	}
	defer q.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var result any
	switch os.Args[1] {
	case "stats":
		result, err = q.Stats(ctx, kind)
	case "dead":
		var items []queue.Item
		items, err = q.DeadLetters(ctx, kind)
		if err == nil {
			dead := make([]map[string]string, 0, len(items))
			for _, item := range items {
				dead = append(dead, map[string]string{"id": item.ID, "payload": string(item.Payload), "failure": item.Failure})
			}
			result = dead
		}
	case "replay":
		if len(os.Args) != 4 {
			fail(fmt.Errorf("replay requires an id"))
		}
		err = q.ReplayDeadLetter(ctx, kind, os.Args[3])
		result = map[string]string{"replayed": os.Args[3]}
	default:
		fail(fmt.Errorf("unknown operation %q", os.Args[1]))
	}
	if err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
