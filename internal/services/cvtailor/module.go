package cvtailor

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/letterwrite"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

const tickInterval = 500 * time.Millisecond

// Deps are the stores and collaborators Build wires into the Module; New
// builds the real ones and calls Build (ADR 0012).
type Deps struct {
	Store   Store
	Docs    DocFetcher
	Drive   Drive
	Asker   Asker
	Cached  CachedAsker
	Editor  Editor
	Lines   *lineedit.Editor
	Profile ProfileWriter
	Creds   Credentials
	Log     *slog.Logger

	Scaffolder Scaffolder
}

type ProfileWriter interface {
	Redraft(ctx context.Context, apiKey string, in profiledraft.Input) (profiledraft.Result, error)
}

type Module struct {
	profile    ProfileWriter
	store      Store
	svc        *Service
	docs       DocFetcher
	drive      Drive
	editor     Editor
	lines      *lineedit.Editor
	creds      Credentials
	cached     CachedAsker
	scaffolder Scaffolder
	log        *slog.Logger
}

func Build(deps Deps) *Module {
	log := deps.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Module{
		log:     log,
		store:   deps.Store,
		svc:     NewService(deps.Store, deps.Docs, deps.Asker, deps.Drive),
		docs:    deps.Docs,
		drive:   deps.Drive,
		editor:  deps.Editor,
		lines:   deps.Lines,
		profile: deps.Profile,
		creds:   deps.Creds,
		cached:  deps.Cached,

		scaffolder: deps.Scaffolder,
	}
}

func New(pool *pgxpool.Pool, docs Docs, asker Asker, cached CachedAsker, creds Credentials, log *slog.Logger) *Module {
	return Build(Deps{Store: store.New(pool), Docs: docs, Drive: docs, Asker: asker, Cached: cached, Editor: cvedit.NewClient(), Lines: lineedit.New(lineedit.NewOpenRouter()), Profile: profiledraft.New(lineedit.NewOpenRouter()), Creds: creds, Log: log, Scaffolder: letterwrite.NewClient()})
}

func NewFacade(pool *pgxpool.Pool) *Module {
	return Build(Deps{Store: store.New(pool)})
}

// Run generates pending Drafts until ctx is cancelled.
func (m *Module) Run(ctx context.Context) error {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := m.RunTick(ctx); err != nil && ctx.Err() == nil {
				slog.ErrorContext(ctx, "draft tick failed", slog.Any(logger.KeyErr, err))
			}
		}
	}
}
