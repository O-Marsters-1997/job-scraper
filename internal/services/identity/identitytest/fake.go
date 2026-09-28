// Package identitytest is a map-backed fake of identity.Store, proven
// against the real store by RunStoreContract (ADR 0012).
package identitytest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

// FakeStore backs identity.Deps.Store; its CreateUser/UpdateEmail and AI
// credential methods also satisfy profile.Store and aicredentials.Store, so
// one fake covers every store Deps field.
type FakeStore struct {
	mu       sync.Mutex
	users    map[string]dto.User
	byName   map[string]string
	sessions map[string]dto.Session
	aiCreds  map[string]string
}

func NewFakeStore() *FakeStore {
	return &FakeStore{
		users:    make(map[string]dto.User),
		byName:   make(map[string]string),
		sessions: make(map[string]dto.Session),
		aiCreds:  make(map[string]string),
	}
}

func (f *FakeStore) GetUserByUsername(_ context.Context, username string) (dto.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byName[username]
	if !ok {
		return dto.User{}, data.ErrNotFound
	}
	return f.users[id], nil
}

func (f *FakeStore) CreateUser(_ context.Context, username, passwordHash, email string) (dto.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createUser(username, passwordHash, email)
}

func (f *FakeStore) CreateUserTx(_ context.Context, _ pgx.Tx, username, passwordHash, email string) (dto.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createUser(username, passwordHash, email)
}

func (f *FakeStore) createUser(username, passwordHash, email string) (dto.User, error) {
	if _, exists := f.byName[username]; exists {
		return dto.User{}, store.ErrUsernameTaken
	}
	u := dto.User{ID: fmt.Sprintf("user-%d", len(f.users)+1), Username: username, PasswordHash: passwordHash, Email: email}
	f.users[u.ID] = u
	f.byName[username] = u.ID
	return u, nil
}

func (f *FakeStore) Begin(context.Context) (pgx.Tx, error) { return fakeTx{}, nil }

func (f *FakeStore) CreateSession(_ context.Context, userID string, expiresAt time.Time) (dto.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := dto.Session{ID: fmt.Sprintf("session-%d", len(f.sessions)+1), UserID: userID, ExpiresAt: expiresAt}
	f.sessions[s.ID] = s
	return s, nil
}

func (f *FakeStore) GetSession(_ context.Context, id string) (dto.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[id]
	if !ok || !s.ExpiresAt.After(time.Now()) {
		return dto.Session{}, data.ErrNotFound
	}
	s.Username = f.users[s.UserID].Username
	return s, nil
}

func (f *FakeStore) DeleteSession(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, id)
	return nil
}

func (f *FakeStore) DeleteExpiredSessions(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	for id, s := range f.sessions {
		if !s.ExpiresAt.After(now) {
			delete(f.sessions, id)
		}
	}
	return nil
}

func (f *FakeStore) GetProfile(_ context.Context, userID string) (dto.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return dto.Profile{}, data.ErrNotFound
	}
	return dto.Profile{Username: u.Username, Email: u.Email}, nil
}

func (f *FakeStore) UpdateEmail(_ context.Context, userID, email string) (dto.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return dto.Profile{}, data.ErrNotFound
	}
	u.Email = email
	f.users[userID] = u
	return dto.Profile{Username: u.Username, Email: u.Email}, nil
}

func (f *FakeStore) UpsertUserAICredential(_ context.Context, userID, provider, encKey string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.aiCreds[userID+"/"+provider] = encKey
	return nil
}

func (f *FakeStore) GetUserAICredential(_ context.Context, userID, provider string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	enc, ok := f.aiCreds[userID+"/"+provider]
	if !ok {
		return "", data.ErrNotFound
	}
	return enc, nil
}

func (f *FakeStore) DeleteUserAICredential(_ context.Context, userID, provider string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.aiCreds, userID+"/"+provider)
	return nil
}

func (f *FakeStore) ListUserAICredentialProviders(_ context.Context, userID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prefix := userID + "/"
	var out []string
	for k := range f.aiCreds {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, k[len(prefix):])
		}
	}
	return out, nil
}

var _ identity.Store = (*FakeStore)(nil)

type fakeTx struct{}

func (fakeTx) Begin(context.Context) (pgx.Tx, error) { return fakeTx{}, nil }
func (fakeTx) Commit(context.Context) error          { return nil }
func (fakeTx) Rollback(context.Context) error        { return nil }
func (fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (fakeTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (fakeTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (fakeTx) Conn() *pgx.Conn                                         { return nil }
