package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func fromSourceTarget(row pgsqlc.SourceTarget) dto.SourceTarget {
	filters := map[string]string{}
	if len(row.Filters) > 0 {
		_ = json.Unmarshal(row.Filters, &filters)
	}
	t := dto.SourceTarget{
		ID:                   row.ID.String(),
		UserID:               row.UserID.String(),
		Source:               row.Source,
		Value:                row.Value,
		Enabled:              row.Enabled,
		Filters:              filters,
		CheckIntervalMinutes: int(row.CheckIntervalMinutes),
		RunStatus:            row.RunStatus,
		LastRunError:         row.LastRunError,
		UpdatedAt:            row.UpdatedAt.Time,
	}
	if row.RunID.Valid {
		t.RunID = row.RunID.String()
	}
	if row.CompanyID.Valid {
		t.CompanyID = row.CompanyID.String()
	}
	if row.LastCheckedAt.Valid {
		lc := row.LastCheckedAt.Time
		t.LastCheckedAt = &lc
	}
	if row.LastRunAt.Valid {
		lastRun := row.LastRunAt.Time
		t.LastRunAt = &lastRun
	}
	return t
}

func (db *DB) SetSourceTargetRunState(ctx context.Context, id, status, runError string) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.SetSourceTargetRunState(ctx, pgsqlc.SetSourceTargetRunStateParams{
		ID: tid, RunStatus: status, LastRunError: runError,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.SourceTarget{}, data.ErrNotFound
		}
		return dto.SourceTarget{}, fmt.Errorf("db.SetSourceTargetRunState: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) StartSourceTargetRun(ctx context.Context, id string) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.StartSourceTargetRun(ctx, tid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("start source target run: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) TransitionSourceTargetRun(ctx context.Context, id, runID, status, runError string) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	rid, err := parseUUID(runID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.TransitionSourceTargetRun(ctx, pgsqlc.TransitionSourceTargetRunParams{
		ID: tid, RunID: rid, RunStatus: status, LastRunError: runError,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("transition source target run: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) GetSourceTarget(ctx context.Context, id string) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.GetSourceTarget(ctx, tid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	if err != nil {
		return dto.SourceTarget{}, err
	}
	return fromSourceTarget(row), nil
}

func (db *DB) ListRecoverableSourceTargets(ctx context.Context) ([]dto.SourceTarget, error) {
	rows, err := db.queries.ListRecoverableSourceTargets(ctx)
	if err != nil {
		return nil, err
	}
	return fromSourceTargets(rows), nil
}

func (db *DB) ClaimRecoverableSourceTarget(ctx context.Context, id, runID string) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	rid, err := parseUUID(runID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.ClaimRecoverableSourceTarget(ctx, pgsqlc.ClaimRecoverableSourceTargetParams{ID: tid, RunID: rid})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("claim recoverable source target: %w", err)
	}
	return fromSourceTarget(row), nil
}

func fromSourceTargets(rows []pgsqlc.SourceTarget) []dto.SourceTarget {
	out := make([]dto.SourceTarget, len(rows))
	for i, r := range rows {
		out[i] = fromSourceTarget(r)
	}
	return out
}

func (db *DB) ListSourceTargetsByUser(ctx context.Context, userID string) ([]dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListSourceTargetsByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.ListSourceTargetsByUser: %w", err)
	}
	return fromSourceTargets(rows), nil
}

func (db *DB) ListEnabledSourceTargets(ctx context.Context) ([]dto.SourceTarget, error) {
	rows, err := db.queries.ListEnabledSourceTargets(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.ListEnabledSourceTargets: %w", err)
	}
	return fromSourceTargets(rows), nil
}

func (db *DB) CreateSourceTarget(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("db.CreateSourceTarget: marshal filters: %w", err)
	}
	row, err := db.queries.CreateSourceTarget(ctx, pgsqlc.CreateSourceTargetParams{
		UserID:  uid,
		Source:  source,
		Value:   value,
		Enabled: enabled,
		Filters: filtersJSON,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.SourceTarget{}, providers.ErrSourceTargetExists
		}
		return dto.SourceTarget{}, fmt.Errorf("db.CreateSourceTarget: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) CreateSourceTargetWithRun(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.CreateSourceTargetWithRun(ctx, pgsqlc.CreateSourceTargetWithRunParams{
		UserID: uid, Source: source, Value: value, Enabled: enabled, Filters: filtersJSON,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.SourceTarget{}, providers.ErrSourceTargetExists
		}
		return dto.SourceTarget{}, fmt.Errorf("create source target with run: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) UpdateSourceTarget(ctx context.Context, id, userID string, enabled *bool, checkIntervalMinutes *int) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	params := pgsqlc.UpdateSourceTargetParams{ID: tid, UserID: uid}
	if enabled != nil {
		params.Enabled = pgtype.Bool{Bool: *enabled, Valid: true}
	}
	if checkIntervalMinutes != nil {
		params.CheckIntervalMinutes = pgtype.Int4{Int32: int32(*checkIntervalMinutes), Valid: true}
	}
	row, err := db.queries.UpdateSourceTarget(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.SourceTarget{}, data.ErrNotFound
		}
		return dto.SourceTarget{}, fmt.Errorf("db.UpdateSourceTarget: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) ListDueSourceTargets(ctx context.Context) ([]dto.SourceTarget, error) {
	rows, err := db.queries.ListDueSourceTargets(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.ListDueSourceTargets: %w", err)
	}
	return fromSourceTargets(rows), nil
}

func (db *DB) TouchSourceTargetsChecked(ctx context.Context, source, value string) error {
	if err := db.queries.TouchSourceTargetsChecked(ctx, pgsqlc.TouchSourceTargetsCheckedParams{
		Source: source,
		Value:  value,
	}); err != nil {
		return fmt.Errorf("db.TouchSourceTargetsChecked: %w", err)
	}
	return nil
}

func (db *DB) UpsertSourceTargetForCompany(ctx context.Context, userID, source, value, companyID string, enabled bool, interval int) (dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	cid, err := parseUUID(companyID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.UpsertSourceTargetForCompany(ctx, pgsqlc.UpsertSourceTargetForCompanyParams{
		UserID:               uid,
		Source:               source,
		Value:                value,
		Enabled:              enabled,
		CompanyID:            cid,
		CheckIntervalMinutes: int32(interval),
	})
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("db.UpsertSourceTargetForCompany: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) ListUserIDsForTarget(ctx context.Context, source, value string) ([]string, error) {
	uuids, err := db.queries.ListUserIDsForTarget(ctx, pgsqlc.ListUserIDsForTargetParams{
		Source: source,
		Value:  value,
	})
	if err != nil {
		return nil, fmt.Errorf("db.ListUserIDsForTarget: %w", err)
	}
	ids := make([]string, len(uuids))
	for i, u := range uuids {
		ids[i] = u.String()
	}
	return ids, nil
}

func (db *DB) DeleteSourceTarget(ctx context.Context, id, userID string) error {
	tid, err := parseUUID(id)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.DeleteSourceTarget(ctx, pgsqlc.DeleteSourceTargetParams{
		ID:     tid,
		UserID: uid,
	}); err != nil {
		return fmt.Errorf("db.DeleteSourceTarget: %w", err)
	}
	return nil
}
