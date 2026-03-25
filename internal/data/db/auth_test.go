package db_test

import (
	"context"
	"testing"
	"time"
)

func truncateAuth(t *testing.T) {
	t.Helper()
	if _, err := testDB.Pool().Exec(context.Background(), "TRUNCATE sessions, users CASCADE"); err != nil {
		t.Fatalf("truncate auth tables: %v", err)
	}
}

func TestCreateUser_And_GetUserByUsername(t *testing.T) {
	truncateAuth(t)
	ctx := context.Background()

	created, err := testDB.CreateUser(ctx, "alice", "hashed-password")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.Username != "alice" {
		t.Errorf("username: got %q, want %q", created.Username, "alice")
	}
	if created.PasswordHash != "hashed-password" {
		t.Errorf("password_hash: got %q, want %q", created.PasswordHash, "hashed-password")
	}
	if created.ID == "" {
		t.Error("ID should not be empty")
	}

	got, err := testDB.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("id mismatch: got %q, want %q", got.ID, created.ID)
	}
	if got.Username != "alice" {
		t.Errorf("username: got %q, want %q", got.Username, "alice")
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	truncateAuth(t)
	ctx := context.Background()

	_, err := testDB.GetUserByUsername(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
}

func TestCreateSession_And_GetSession(t *testing.T) {
	truncateAuth(t)
	ctx := context.Background()

	user, err := testDB.CreateUser(ctx, "bob", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC()
	session, err := testDB.CreateSession(ctx, user.ID, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
	if session.UserID != user.ID {
		t.Errorf("user_id: got %q, want %q", session.UserID, user.ID)
	}

	got, err := testDB.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.ID != session.ID {
		t.Errorf("id: got %q, want %q", got.ID, session.ID)
	}
	if got.Username != "bob" {
		t.Errorf("username: got %q, want %q", got.Username, "bob")
	}
}

func TestGetSession_Expired(t *testing.T) {
	truncateAuth(t)
	ctx := context.Background()

	user, err := testDB.CreateUser(ctx, "charlie", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Create session that expired 1 second ago.
	expiredAt := time.Now().Add(-1 * time.Second)
	session, err := testDB.CreateSession(ctx, user.ID, expiredAt)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	_, err = testDB.GetSession(ctx, session.ID)
	if err == nil {
		t.Fatal("expected error for expired session, got nil")
	}
}

func TestDeleteSession(t *testing.T) {
	truncateAuth(t)
	ctx := context.Background()

	user, err := testDB.CreateUser(ctx, "dana", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	session, err := testDB.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := testDB.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	_, err = testDB.GetSession(ctx, session.ID)
	if err == nil {
		t.Fatal("expected error after deletion, got nil")
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	truncateAuth(t)
	ctx := context.Background()

	user, err := testDB.CreateUser(ctx, "eve", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// One valid session.
	valid, err := testDB.CreateSession(ctx, user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession (valid): %v", err)
	}

	// One expired session — insert directly via pool to bypass the expires_at > NOW() check.
	_, err = testDB.Pool().Exec(ctx,
		"INSERT INTO sessions (user_id, expires_at) VALUES ($1, $2)",
		user.ID, time.Now().Add(-time.Second),
	)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	if err := testDB.DeleteExpiredSessions(ctx); err != nil {
		t.Fatalf("DeleteExpiredSessions: %v", err)
	}

	// Valid session should still exist.
	if _, err := testDB.GetSession(ctx, valid.ID); err != nil {
		t.Errorf("valid session should survive cleanup: %v", err)
	}

	// Expired row count should be zero.
	var count int
	if err := testDB.Pool().QueryRow(ctx,
		"SELECT COUNT(*) FROM sessions WHERE expires_at <= NOW()",
	).Scan(&count); err != nil {
		t.Fatalf("count expired: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 expired sessions after cleanup, got %d", count)
	}
}
