package proxy

import "time"

func SetRetryBackoff(d time.Duration) (restore func()) {
	prev := retryBackoff
	retryBackoff = d
	return func() { retryBackoff = prev }
}
