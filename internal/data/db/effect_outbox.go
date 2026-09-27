package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
)

func queueAnswerEffect(ctx context.Context, queries *pgsqlc.Queries, jobID pgtype.UUID, fingerprint string, discovery, firstDiscovery bool) error {
	return queries.QueueAnswerEffect(ctx, pgsqlc.QueueAnswerEffectParams{
		JobID: jobID, Fingerprint: fingerprint, Discovery: discovery, FirstDiscovery: firstDiscovery,
	})
}
