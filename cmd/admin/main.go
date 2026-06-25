package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/data"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

func main() {
	slog.SetDefault(logger.New())

	if len(os.Args) < 3 || os.Args[1] != "create-user" {
		fmt.Fprintln(os.Stderr, "usage: admin create-user <username>")
		os.Exit(1)
	}

	username := os.Args[2]

	fmt.Fprint(os.Stderr, "Password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read password: %v\n", err)
		os.Exit(1)
	}
	if len(passwordBytes) == 0 {
		fmt.Fprintln(os.Stderr, "password must not be empty")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword(passwordBytes, 12)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	connStr, err := jobsdb.ConnString()
	if err != nil {
		fmt.Fprintf(os.Stderr, "db config invalid: %v\n", err)
		os.Exit(1)
	}
	db, err := jobsdb.New(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := data.RunMigrations(ctx, db.Pool()); err != nil {
		fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
		os.Exit(1)
	}

	user, err := db.CreateUser(ctx, username, string(hash), "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User %q created (id: %s)\n", user.Username, user.ID)
}
