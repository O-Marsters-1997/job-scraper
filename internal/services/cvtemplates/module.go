package cvtemplates

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/store"
	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs"
)

type Module struct {
	cv          *Service
	trackedDocs *trackeddocs.Service
}

func New(pool *pgxpool.Pool, gc DocsClient) *Module {
	st := store.New(pool)
	return &Module{
		cv:          NewService(gc, st),
		trackedDocs: trackeddocs.New(gc, st),
	}
}
