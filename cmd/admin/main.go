package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/data/db"
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
	fmt.Fprintln(os.Stderr, "       admin scoring-feedback export <username>")
	fmt.Fprintln(os.Stderr, "       admin scoring-feedback clear <username>")
	os.Exit(1)
}

func main() {
	slog.SetDefault(logger.MustFromEnv())

	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "create-user":
		runCreateUser(os.Args[2:])
	case "options":
		runOptions(os.Args[2:])
	case "scoring-feedback":
		runScoringFeedback(os.Args[2:])
	default:
		usage()
	}
}

func connectDB(ctx context.Context) *pgxpool.Pool {
	pool, err := db.Connect(ctx)
	if err != nil {
		fatal("db connect", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		fatal("run migrations", err)
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
		fatal("read password", err)
	}
	if len(passwordBytes) == 0 {
		fmt.Fprintln(os.Stderr, "password must not be empty")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword(passwordBytes, 12)
	if err != nil {
		fatal("hash password", err)
	}

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()

	apps := applications.New(pool)
	idm := identity.NewFacade(pool, apps)
	user, err := idm.CreateUser(ctx, dto.CreateUserInput{Username: username, PasswordHash: string(hash)})
	if err != nil {
		fatal("create user", err)
	}

	fmt.Printf("User %q created (id: %s)\n", user.Username, user.ID)
}

func runOptions(args []string) {
	if len(args) < 1 {
		usage()
	}
	var run func(context.Context, *scoring.Module) error
	var label, done string
	switch args[0] {
	case "add":
		if len(args) != 5 {
			usage()
		}
		id, dimension, optionLabel, question := args[1], args[2], args[3], args[4]
		run = func(ctx context.Context, m *scoring.Module) error {
			return m.AddOption(ctx, id, dimension, optionLabel, question)
		}
		label, done = "add option", fmt.Sprintf("Option %q added", id)
	case "reword":
		if len(args) != 3 {
			usage()
		}
		id, question := args[1], args[2]
		run = func(ctx context.Context, m *scoring.Module) error { return m.RewordOption(ctx, id, question) }
		label, done = "reword option", fmt.Sprintf("Option %q reworded", id)
	case "retire":
		if len(args) != 2 {
			usage()
		}
		id := args[1]
		run = func(ctx context.Context, m *scoring.Module) error { return m.RetireOption(ctx, id) }
		label, done = "retire option", fmt.Sprintf("Option %q retired", id)
	default:
		usage()
	}

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()
	if err := run(ctx, scoring.NewFacade(pool)); err != nil {
		fatal(label, err)
	}
	fmt.Println(done)
}

func runScoringFeedback(args []string) {
	if len(args) != 2 || (args[0] != "export" && args[0] != "clear") {
		usage()
	}
	action, username := args[0], args[1]

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()

	userID, err := identity.NewFacade(pool, nil).UserIDByUsername(ctx, username)
	if err != nil {
		fatal("find user", err)
	}
	m := scoring.NewFacade(pool)
	if action == "clear" {
		n, err := m.ClearFeedback(ctx, userID)
		if err != nil {
			fatal("clear feedback", err)
		}
		fmt.Fprintf(os.Stderr, "Cleared %d entries\n", n)
		return
	}
	pack, err := m.ExportFeedback(ctx, userID)
	if err != nil {
		fatal("export feedback", err)
	}
	fmt.Print(pack)
}

func fatal(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	os.Exit(1)
}
