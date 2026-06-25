package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	app "github.com/ollymarsters/job-scraper/internal"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

func main() {
	slog.SetDefault(logger.New())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	connStr, err := jobsdb.ConnString()
	if err != nil {
		slog.Error("db config invalid", slog.Any("err", err))
		os.Exit(1)
	}
	db, err := jobsdb.New(ctx, connStr)
	if err != nil {
		slog.Error("db init failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer db.Close()

	cs, err := credstore.New(db)
	if err != nil {
		slog.Error("credstore init failed", slog.Any("err", err))
		os.Exit(1)
	}

	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "localhost:6379"
	}
	q, err := queue.New(valkeyAddr)
	if err != nil {
		slog.Error("queue init failed", slog.Any("err", err))
		os.Exit(1)
	}
	slog.Info("queue client ready", slog.String("addr", valkeyAddr))
	defer q.Close()

	port := os.Getenv("API_PORT")
	if port == "" {
		port = ":8080"
	}

	srv := &http.Server{Addr: port, Handler: app.NewRouter(ctx, db, q, cs)}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	slog.Info("api server starting", slog.String("addr", port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.Any("err", err))
		os.Exit(1)
	}
}
