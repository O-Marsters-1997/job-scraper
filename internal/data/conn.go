package data

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
)

// ConnString builds the Postgres DSN from POSTGRES_* env vars, shared by
// every binary that connects to the database directly.
func ConnString() (string, error) {
	vars := map[string]string{
		"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
		"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
		"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
		"POSTGRES_PORT":     os.Getenv("POSTGRES_PORT"),
		"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
	}
	var missing []string
	for k, v := range vars {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	sslmode := os.Getenv("POSTGRES_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	dbName := vars["POSTGRES_DB"]
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(vars["POSTGRES_USER"], vars["POSTGRES_PASSWORD"]),
		Host:     net.JoinHostPort(vars["POSTGRES_HOST"], vars["POSTGRES_PORT"]),
		Path:     "/" + dbName,
		RawPath:  "/" + url.PathEscape(dbName),
		RawQuery: url.Values{"sslmode": {sslmode}}.Encode(),
	}).String(), nil
}
