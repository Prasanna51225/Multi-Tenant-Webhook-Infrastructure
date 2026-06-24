package retry

import (
	"math/rand"
	"time"
)

const MaxDelay = 1 * time.Hour

func CalculateBackoff(attempt int, baseMs int) time.Duration {
	base := time.Duration(baseMs) * time.Millisecond
	delay := base

	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay > MaxDelay {
			delay = MaxDelay
			break
		}
	}

	jitter := time.Duration(rand.Int63n(int64(delay)))
	return jitter
}

func NextRetryAt(attempt int, baseMs int) time.Time {
	backoff := CalculateBackoff(attempt, baseMs)
	return time.Now().UTC().Add(backoff)
}
