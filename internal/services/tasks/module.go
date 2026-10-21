// Package tasks is the tasks context: Tasks and their Task types (ADR 0011).
package tasks

import (
	"context"
	"time"

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

func New(db store.DB, applications store.ApplicationSync) *Module {
	return Build(Deps{Store: store.New(db, applications)})
}

func NewFacade(db store.DB) *Module {
	return New(db, nil)
}

// SeedDefaults seeds userID's default Task types inside tx; identity's signup
// calls this through its Seeder interface (ADR 0011).
func (m *Module) SeedDefaults(ctx context.Context, tx pgx.Tx, userID string) error {
	return m.store.SeedDefaultTaskTypes(ctx, tx, userID)
}

func (m *Module) ApplicationTracked(ctx context.Context, tx pgx.Tx, userID, jobID string) error {
	return m.store.CreateApplyTask(ctx, tx, userID, jobID)
}

func (m *Module) ApplicationApplied(ctx context.Context, tx pgx.Tx, userID, jobID string) error {
	return m.store.CompleteApplyTasks(ctx, tx, userID, jobID, time.Now())
}
