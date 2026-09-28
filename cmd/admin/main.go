package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: admin create-user <username>")
	fmt.Fprintln(os.Stderr, "       admin options add <id> <dimension> <label> <question>")
	fmt.Fprintln(os.Stderr, "       admin options reword <id> <question>")
	fmt.Fprintln(os.Stderr, "       admin options retire <id>")
	os.Exit(1)
}

func main() {
	slog.SetDefault(logger.New())

	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "create-user":
		runCreateUser(os.Args[2:])
	case "options":
		runOptions(os.Args[2:])
	default:
		usage()
	}
}

func connectDB(ctx context.Context) *pgxpool.Pool {
	pool, err := data.Connect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect: %v\n", err)
		os.Exit(1)
	}
	if err := data.RunMigrations(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
		os.Exit(1)
	}
	return pool
}

func runCreateUser(args []string) {
	if len(args) != 1 {
		usage()
	}
	username := args[0]

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
	pool := connectDB(ctx)
	defer pool.Close()

	apps := applications.New(pool)
	idm := identity.NewFacade(pool, apps)
	user, err := idm.CreateUser(ctx, dto.CreateUserInput{Username: username, PasswordHash: string(hash)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User %q created (id: %s)\n", user.Username, user.ID)
}

func runOptions(args []string) {
	if len(args) < 1 {
		usage()
	}
	switch args[0] {
	case "add":
		if len(args) != 5 {
			usage()
		}
	case "reword":
		if len(args) != 3 {
			usage()
		}
	case "retire":
		if len(args) != 2 {
			usage()
		}
	default:
		usage()
	}

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()
	scoringModule := scoring.NewFacade(pool)

	switch args[0] {
	case "add":
		id, dimension, label, question := args[1], args[2], args[3], args[4]
		if err := scoringModule.AddOption(ctx, id, dimension, label, question); err != nil {
			fmt.Fprintf(os.Stderr, "add option: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Option %q added\n", id)
	case "reword":
		id, question := args[1], args[2]
		if err := scoringModule.RewordOption(ctx, id, question); err != nil {
			fmt.Fprintf(os.Stderr, "reword option: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Option %q reworded\n", id)
	case "retire":
		id := args[1]
		if err := scoringModule.RetireOption(ctx, id); err != nil {
			fmt.Fprintf(os.Stderr, "retire option: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Option %q retired\n", id)
	}
}
