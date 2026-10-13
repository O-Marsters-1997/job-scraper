package identity

import (
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
)

const minPasswordBytes, maxPasswordBytes = 8, 72

func validatePassword(p string) error {
	if n := len(p); n < minPasswordBytes || n > maxPasswordBytes {
		return apperr.Invalid(fmt.Sprintf("password must be %d to %d bytes", minPasswordBytes, maxPasswordBytes))
	}
	return nil
}
