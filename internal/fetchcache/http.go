package fetchcache

import (
	"fmt"
	"time"
)

const (
	notFoundTTL = time.Hour
	errorTTL    = 2 * time.Minute
)

// StatusError reports an unsuccessful HTTP status from an upstream source.
type StatusError struct {
	Code   int
	Source string
}

func (e *StatusError) Error() string { return fmt.Sprintf("HTTP %d from %s", e.Code, e.Source) }

// DocTTL is a ready-made Options.TTL for caches of downloaded documents:
// successes live for ok, a 404 or 410 is remembered for an hour so a dead
// file is not retried for every channel, and other failures for two minutes.
func DocTTL[V any](ok time.Duration) func(V, error) time.Duration {
	return func(_ V, err error) time.Duration {
		if err == nil {
			return ok
		}
		if se, ok := err.(*StatusError); ok && (se.Code == 404 || se.Code == 410) {
			return notFoundTTL
		}
		return errorTTL
	}
}
