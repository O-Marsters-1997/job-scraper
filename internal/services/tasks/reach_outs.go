package tasks

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type ReachOuts interface {
	Open(ctx context.Context, tx pgx.Tx, userID, jobID, contactID string) (taskID string, err error)
	CompleteForContact(ctx context.Context, tx pgx.Tx, userID, contactID string) error
}

var _ ReachOuts = (*Module)(nil)

func (m *Module) Open(ctx context.Context, tx pgx.Tx, userID, jobID, contactID string) (string, error) {
	return m.store.OpenReachOutTask(ctx, tx, userID, jobID, contactID)
}

func (m *Module) CompleteForContact(ctx context.Context, tx pgx.Tx, userID, contactID string) error {
	return m.store.CompleteReachOutTasksForContact(ctx, tx, userID, contactID, time.Now())
}
