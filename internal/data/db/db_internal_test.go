package db

import (
	"net/url"
	"strings"
	"testing"
)

func TestConnString(t *testing.T) {
	for _, key := range []string{"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_DB"} {
		t.Setenv(key, "")
	}
	_, err := ConnString()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_DB, POSTGRES_HOST, POSTGRES_PASSWORD, POSTGRES_PORT, POSTGRES_USER") {
		t.Fatalf("missing variables: %v", err)
	}
	t.Setenv("POSTGRES_USER", "user@name")
	t.Setenv("POSTGRES_PASSWORD", "pass/word?#")
	t.Setenv("POSTGRES_HOST", "::1")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "jobs/archive")
	t.Setenv("POSTGRES_SSLMODE", "require")
	conn, err := ConnString()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(conn)
	if err != nil {
		t.Fatal(err)
	}
	pass, _ := u.User.Password()
	if u.User.Username() != "user@name" || pass != "pass/word?#" || u.Host != "[::1]:5432" || u.EscapedPath() != "/jobs%2Farchive" || u.Query().Get("sslmode") != "require" {
		t.Fatalf("bad connection URL: %s", conn)
	}
}
