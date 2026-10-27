package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/term"

	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/events"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/tasks"
	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: admin create-user <username>")
	fmt.Fprintln(os.Stderr, "       admin reset-password <username>")
	fmt.Fprintln(os.Stderr, "       admin set-role <username> <user|admin>")
	fmt.Fprintln(os.Stderr, "       admin reencrypt")
	fmt.Fprintln(os.Stderr, "       admin options add <id> <dimension> <label> <question>")
	fmt.Fprintln(os.Stderr, "       admin options reword <id> <question>")
	fmt.Fprintln(os.Stderr, "       admin options retire <id>")
	fmt.Fprintln(os.Stderr, "       admin scoring-feedback export <username> [--include-outdated]")
	fmt.Fprintln(os.Stderr, "       admin scoring-feedback clear <username>")
	fmt.Fprintln(os.Stderr, "       admin scoring replay <username>")
	fmt.Fprintln(os.Stderr, "       admin scoring fit <username>")
	fmt.Fprintln(os.Stderr, "       admin profile-attempts export <username>")
	fmt.Fprintln(os.Stderr, "       admin resolve-domains <file>")
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
	case "reset-password":
		runResetPassword(os.Args[2:])
	case "set-role":
		runSetRole(os.Args[2:])
	case "reencrypt":
		runReencrypt(os.Args[2:])
	case "options":
		runOptions(os.Args[2:])
	case "scoring-feedback":
		runScoringFeedback(os.Args[2:])
	case "scoring":
		runScoring(os.Args[2:])
	case "profile-attempts":
		runProfileAttempts(os.Args[2:])
	case "resolve-domains":
		runResolveDomains(os.Args[2:])
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

	passwordBytes := readPassword("Password: ")

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()

	taskFacade := tasks.NewFacade(pool)
	apps := applications.New(pool, events.New(pool, scoring.NewFacade(pool)), taskFacade)
	idm := identity.NewFacade(pool, []identity.Seeder{apps, taskFacade})
	user, err := idm.CreateUser(ctx, dto.CreateUserInput{Username: username, Password: string(passwordBytes)})
	if err != nil {
		fatal("create user", err)
	}

	fmt.Printf("User %q created (id: %s)\n", user.Username, user.ID)
}

func runResetPassword(args []string) {
	if len(args) != 1 {
		usage()
	}
	username := args[0]

	password := readPassword("New password: ")
	if string(readPassword("Confirm password: ")) != string(password) {
		fmt.Fprintln(os.Stderr, "passwords do not match")
		os.Exit(1)
	}

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()

	if err := identity.NewFacade(pool, nil).ResetPassword(ctx, username, string(password)); err != nil {
		fatal("reset password", err)
	}
	fmt.Printf("Password for %q reset\n", username)
}

func readPassword(prompt string) []byte {
	fmt.Fprint(os.Stderr, prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fatal("read password", err)
	}
	return password
}

func runSetRole(args []string) {
	if len(args) != 2 {
		usage()
	}
	username, role := args[0], args[1]

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()

	if err := identity.NewFacade(pool, nil).SetRole(ctx, username, role); err != nil {
		fatal("set role", err)
	}
	fmt.Printf("User %q is now %s\n", username, role)
}

func runReencrypt(args []string) {
	if len(args) != 0 {
		usage()
	}
	credCipher, err := tokencrypt.FromEnv("AI_CREDENTIAL_ENC_KEY")
	if err != nil {
		fatal("load credential key", err)
	}
	googleCipher, err := tokencrypt.FromEnv("GOOGLE_TOKEN_ENC_KEY")
	if err != nil {
		fatal("load google token key", err)
	}

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()

	credentials, googleTokens, err := identity.ReencryptTokens(ctx, pool, credCipher, googleCipher)
	if err != nil {
		fatal("reencrypt", err)
	}
	fmt.Printf("Re-encrypted %d AI credentials and %d Google tokens\n", credentials, googleTokens)
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
	if len(args) < 2 || (args[0] != "export" && args[0] != "clear") {
		usage()
	}
	action, username, flags := args[0], args[1], args[2:]
	includeOutdated := action == "export" && len(flags) == 1 && flags[0] == "--include-outdated"
	if len(flags) > 0 && !includeOutdated {
		usage()
	}

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
	pack, err := m.ExportFeedback(ctx, userID, includeOutdated)
	if err != nil {
		fatal("export feedback", err)
	}
	fmt.Print(pack)
}

func runScoring(args []string) {
	if len(args) != 2 || (args[0] != "replay" && args[0] != "fit") {
		usage()
	}

	ctx := context.Background()
	pool := connectDB(ctx)
	defer pool.Close()

	userID, err := identity.NewFacade(pool, nil).UserIDByUsername(ctx, args[1])
	if err != nil {
		fatal("find user", err)
	}
	m := scoring.NewFacade(pool)
	run := m.Replay
	if args[0] == "fit" {
		run = m.Fit
	}
	report, err := run(ctx, userID)
	if err != nil {
		fatal(args[0], err)
	}
	fmt.Print(report)
}

func fatal(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	os.Exit(1)
}
