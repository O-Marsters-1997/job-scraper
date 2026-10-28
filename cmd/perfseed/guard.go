package main

import (
	"errors"
	"fmt"
	"strings"
)

var localHosts = map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true, "db": true}

func checkGuard(env, host string, confirmed bool) error {
	var errs []error
	if env != "dev" {
		errs = append(errs, fmt.Errorf("ALLOY_ENV is %q, want \"dev\"", env))
	}
	if !localHosts[host] && !strings.HasPrefix(host, "/") {
		errs = append(errs, fmt.Errorf("database host %q is not local (localhost, 127.0.0.1, ::1, db or a unix socket)", host))
	}
	if !confirmed {
		errs = append(errs, errors.New("--yes-wipe-dev-data was not passed"))
	}
	return errors.Join(errs...)
}
