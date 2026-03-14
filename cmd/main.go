package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources/greenhouse"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// ── Postgres ─────────────────────────────────────────────────────────────
	connString := postgresConnString()

	db, err := db.New(ctx, connString)
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.Close()

	fmt.Println("connected to postgres")

	// ── Valkey ────────────────────────────────────────────────────────────────
	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "localhost:6379"
	}

	q, err := queue.New(valkeyAddr)
	if err != nil {
		log.Fatalf("queue init: %v", err)
	}
	defer q.Close()

	fmt.Println("connected to valkey")

	// ── Scrape ───────────────────────────────────────────────────────────────
	scraper, err := greenhouse.New(greenhouse.Config{
		BoardTokens: []string{"greenhouse"},
	})
	if err != nil {
		log.Fatal(err)
	}

	jobs, err := scraper.FetchJobs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scrape error: %v\n", err)
	}

	fmt.Printf("scraped %d jobs from %s\n", len(jobs), scraper.Name())

	// ── Deduplicate ──────────────────────────────────────────────────────────
	seen := make(map[string]struct{}, len(jobs))
	var urls []string
	for _, j := range jobs {
		if j.URL == "" {
			continue
		}
		if _, ok := seen[j.URL]; ok {
			continue
		}
		seen[j.URL] = struct{}{}
		urls = append(urls, j.URL)
	}

	fmt.Printf("unique urls: %d\n", len(urls))

	// ── Enqueue ──────────────────────────────────────────────────────────────
	if err := q.Enqueue(ctx, urls, time.Now()); err != nil {
		log.Fatalf("enqueue: %v", err)
	}

	fmt.Printf("enqueued %d urls into jobs:pending\n", len(urls))

	_ = db // db will be used by repository layer once added
}

func postgresConnString() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	host := envOr("POSTGRES_HOST", "localhost")
	port := envOr("POSTGRES_PORT", "5433")
	user := envOr("POSTGRES_USER", "postgres")
	pass := envOr("POSTGRES_PASSWORD", "postgres")
	name := envOr("POSTGRES_DB", "job_scraper")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, name)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
