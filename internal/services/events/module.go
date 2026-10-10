// Package events is the events context: the generic, append-only events log
// (ADR 0011, ADR 0025).
package events

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/services/events/store"
)

type Deps struct {
	Store     Store
	Snapshots ScoreSnapshotter
}

type Module struct {
	service *Service
}

func Build(deps Deps) *Module {
	return &Module{service: NewService(deps.Store, deps.Snapshots)}
}

func New(pool *pgxpool.Pool, snapshots ScoreSnapshotter) *Module {
	return Build(Deps{Store: store.New(pool), Snapshots: snapshots})
}

func (m *Module) Routes(r chi.Router) {
	r.Post("/events", handlers.Create(m.service.Record))
}

// RecordApplicationCreated logs the creation of applicationID for jobID
// inside tx, so the event commits or rolls back with the application.
func (m *Module) RecordApplicationCreated(ctx context.Context, tx pgx.Tx, userID, applicationID, jobID string) error {
	return m.service.recordApplicationEvent(ctx, tx, userID, ApplicationCreated, applicationID, applicationProps{JobID: jobID})
}

// RecordApplicationStatusChanged logs applicationID moving from one status to
// another inside tx.
func (m *Module) RecordApplicationStatusChanged(ctx context.Context, tx pgx.Tx, userID, applicationID, jobID, fromStatusID, toStatusID string) error {
	return m.service.recordApplicationEvent(ctx, tx, userID, ApplicationStatusChanged, applicationID,
		applicationProps{JobID: jobID, FromStatusID: fromStatusID, ToStatusID: toStatusID})
}
