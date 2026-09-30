package cvtemplates

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/store"
)

// Deps are the stores and collaborators Build wires into the Module; New
// builds the real ones and calls Build (ADR 0012).
type Deps struct {
	Store      Store
	DocsClient DocsClient
}

type Module struct {
	svc   *Service
	store Store
}

func Build(deps Deps) *Module {
	return &Module{svc: NewService(deps.DocsClient, deps.Store), store: deps.Store}
}

func New(pool *pgxpool.Pool, gc DocsClient) *Module {
	return Build(Deps{Store: store.New(pool), DocsClient: gc})
}
