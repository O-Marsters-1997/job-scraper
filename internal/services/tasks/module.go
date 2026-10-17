// Package tasks is the tasks context: Tasks and their Task types (ADR 0011).
package tasks

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/services/tasks/store"
)

type Deps struct {
	Store Store
}

type Module struct {
	service *Service
	store   Store
}

func Build(deps Deps) *Module {
	return &Module{service: NewService(deps.Store), store: deps.Store}
}

func New(db store.DBTX) *Module {
	return Build(Deps{Store: store.New(db)})
}

// SeedDefaults seeds userID's default Task types inside tx; identity's signup
// calls this through its Seeder interface (ADR 0011).
func (m *Module) SeedDefaults(ctx context.Context, tx pgx.Tx, userID string) error {
	return m.store.SeedDefaultTaskTypes(ctx, tx, userID)
}
