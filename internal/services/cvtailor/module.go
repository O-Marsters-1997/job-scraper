package cvtailor

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

const tickInterval = 2 * time.Second

// Deps are the stores and collaborators Build wires into the Module; New
// builds the real ones and calls Build (ADR 0012).
type Deps struct {
	Store  Store
	Docs   DocFetcher
	Drive  Drive
	Asker  Asker
	Editor Editor
	Creds  Credentials
}

type Module struct {
	store  Store
	svc    *Service
	docs   DocFetcher
	drive  Drive
	editor Editor
	creds  Credentials
}

func Build(deps Deps) *Module {
	return &Module{
		store:  deps.Store,
		svc:    NewService(deps.Store, deps.Docs, deps.Asker, deps.Drive),
		docs:   deps.Docs,
		drive:  deps.Drive,
		editor: deps.Editor,
		creds:  deps.Creds,
	}
}

func New(pool *pgxpool.Pool, docs Docs, asker Asker, creds Credentials) *Module {
	return Build(Deps{Store: store.New(pool), Docs: docs, Drive: docs, Asker: asker, Editor: cvedit.NewClient(), Creds: creds})
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
