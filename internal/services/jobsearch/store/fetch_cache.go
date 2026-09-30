package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
)

func (s *Store) LookupFetch(ctx context.Context, url string) (dto.CachedResponse, bool, error) {
	row, err := s.queries.GetFetchCache(ctx, url)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.CachedResponse{}, false, nil
	}
	if err != nil {
		return dto.CachedResponse{}, false, fmt.Errorf("store.LookupFetch: %w", err)
	}
	var header http.Header
	if err := json.Unmarshal(row.Header, &header); err != nil {
		return dto.CachedResponse{}, false, fmt.Errorf("store.LookupFetch: decode header: %w", err)
	}
	return dto.CachedResponse{URL: row.Url, Status: int(row.Status), Header: header, Body: row.Body}, true, nil
}

func (s *Store) PutFetch(ctx context.Context, resp dto.CachedResponse) error {
	header, err := json.Marshal(resp.Header)
	if err != nil {
		return fmt.Errorf("store.PutFetch: %w", err)
	}
	err = s.queries.UpsertFetchCache(ctx, sqlc.UpsertFetchCacheParams{Url: resp.URL, Status: int32(resp.Status), Header: header, Body: resp.Body})
	if err != nil {
		return fmt.Errorf("store.PutFetch: %w", err)
	}
	return nil
}

func (s *Store) ForgetFetches(ctx context.Context, urls []string) error {
	if err := s.queries.DeleteFetchCacheURLs(ctx, urls); err != nil {
		return fmt.Errorf("store.ForgetFetches: %w", err)
	}
	return nil
}

func (s *Store) DeleteExpiredFetches(ctx context.Context) error {
	if err := s.queries.DeleteExpiredFetchCache(ctx); err != nil {
		return fmt.Errorf("store.DeleteExpiredFetches: %w", err)
	}
	return nil
}
