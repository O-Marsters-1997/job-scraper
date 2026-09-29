package cvtailor

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

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
	store Store
	svc   *Service
	gen   *Generator
}

func Build(deps Deps) *Module {
	return &Module{
		store: deps.Store,
		svc:   NewService(deps.Store, deps.Docs, deps.Asker),
		gen: NewGenerator(GeneratorDeps{
			Store: deps.Store, Docs: deps.Docs, Drive: deps.Drive, Editor: deps.Editor, Creds: deps.Creds,
		}),
	}
}

func New(pool *pgxpool.Pool, docs Docs, asker Asker, creds Credentials) *Module {
	return Build(Deps{Store: store.New(pool), Docs: docs, Drive: docs, Asker: asker, Editor: cvedit.NewClient(), Creds: creds})
}

// Run generates pending Drafts until ctx is cancelled.
func (m *Module) Run(ctx context.Context) error {
	return m.gen.Run(ctx)
}
