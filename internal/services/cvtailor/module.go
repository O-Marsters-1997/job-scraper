package cvtailor

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

// Deps are the stores and collaborators Build wires into the Module; New
// builds the real ones and calls Build (ADR 0012).
type Deps struct {
	Store Store
	Docs  DocFetcher
	Asker Asker
}

type Module struct {
	store Store
	svc   *Service
}

func Build(deps Deps) *Module {
	return &Module{store: deps.Store, svc: NewService(deps.Store, deps.Docs, deps.Asker)}
}

func New(pool *pgxpool.Pool, docs DocFetcher, asker Asker) *Module {
	return Build(Deps{Store: store.New(pool), Docs: docs, Asker: asker})
}
