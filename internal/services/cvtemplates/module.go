package cvtemplates

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/store"
	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs"
)

// Deps are the stores and collaborators Build wires into the Module; New
// builds the real ones and calls Build. Tests call Build directly with
// fakes and stubs (ADR 0012).
type Deps struct {
	CV          Store
	TrackedDocs trackeddocs.Store
	DocsClient  DocsClient
}

type Module struct {
	cv          *Service
	trackedDocs *trackeddocs.Service
}

func Build(deps Deps) *Module {
	return &Module{
		cv:          NewService(deps.DocsClient, deps.CV),
		trackedDocs: trackeddocs.New(deps.DocsClient, deps.TrackedDocs),
	}
}

func New(pool *pgxpool.Pool, gc DocsClient) *Module {
	st := store.New(pool)
	return Build(Deps{CV: st, TrackedDocs: st, DocsClient: gc})
}
