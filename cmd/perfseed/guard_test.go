package main

import "testing"

func TestCheckGuard(t *testing.T) {
	t.Parallel()
	t.Run("allows local dev with confirmation", func(t *testing.T) {
		t.Parallel()
		for _, host := range []string{"localhost", "127.0.0.1", "::1", "db", "/var/run/postgresql"} {
			if err := checkGuard("dev", host, true); err != nil {
				t.Errorf("checkGuard(dev, %q, true) = %v, want nil", host, err)
			}
		}
	})

	refusals := []struct {
		name      string
		env, host string
		confirmed bool
	}{
		{"production env", "prod", "localhost", true},
		{"unset env", "", "localhost", true},
		{"remote host", "dev", "db.example.com", true},
		{"remote ip", "dev", "10.0.0.5", true},
		{"no confirmation flag", "dev", "localhost", false},
	}
	for _, tt := range refusals {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := checkGuard(tt.env, tt.host, tt.confirmed); err == nil {
				t.Errorf("checkGuard(%q, %q, %v) = nil, want refusal", tt.env, tt.host, tt.confirmed)
			}
		})
	}
}
