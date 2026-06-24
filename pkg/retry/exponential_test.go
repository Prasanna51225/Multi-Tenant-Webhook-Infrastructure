package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateBackoff(t *testing.T) {
	t.Run("attempt 0 returns jitter around base", func(t *testing.T) {
		backoff := CalculateBackoff(0, 1000)
		assert.GreaterOrEqual(t, backoff, time.Duration(0))
		assert.LessOrEqual(t, backoff, 1000*time.Millisecond)
	})

	t.Run("attempt 1 returns jitter around 2x base", func(t *testing.T) {
		backoff := CalculateBackoff(1, 1000)
		assert.GreaterOrEqual(t, backoff, time.Duration(0))
		assert.LessOrEqual(t, backoff, 2000*time.Millisecond)
	})

	t.Run("attempt 2 returns jitter around 4x base", func(t *testing.T) {
		backoff := CalculateBackoff(2, 1000)
		assert.GreaterOrEqual(t, backoff, time.Duration(0))
		assert.LessOrEqual(t, backoff, 4000*time.Millisecond)
	})

	t.Run("attempt 3 returns jitter around 8x base", func(t *testing.T) {
		backoff := CalculateBackoff(3, 1000)
		assert.GreaterOrEqual(t, backoff, time.Duration(0))
		assert.LessOrEqual(t, backoff, 8000*time.Millisecond)
	})

	t.Run("backoff never exceeds max delay", func(t *testing.T) {
		backoff := CalculateBackoff(20, 1000)
		assert.GreaterOrEqual(t, backoff, time.Duration(0))
		assert.LessOrEqual(t, backoff, MaxDelay)
	})

	t.Run("different calls produce different jitter values", func(t *testing.T) {
		results := make(map[time.Duration]bool)
		for i := 0; i < 100; i++ {
			backoff := CalculateBackoff(3, 1000)
			results[backoff] = true
		}
		assert.Greater(t, len(results), 1, "jitter should produce varied values")
	})

	t.Run("base 500ms works correctly", func(t *testing.T) {
		backoff := CalculateBackoff(1, 500)
		assert.GreaterOrEqual(t, backoff, time.Duration(0))
		assert.LessOrEqual(t, backoff, 1000*time.Millisecond)
	})
}

func TestNextRetryAt(t *testing.T) {
	t.Run("returns time in the future", func(t *testing.T) {
		before := time.Now().UTC()
		retryAt := NextRetryAt(1, 1000)
		after := time.Now().UTC().Add(3 * time.Second)

		assert.True(t, retryAt.After(before) || retryAt.Equal(before))
		assert.True(t, retryAt.Before(after) || retryAt.Equal(after))
	})

	t.Run("higher attempt returns later time on average", func(t *testing.T) {
		retry1 := NextRetryAt(1, 1000)
		retry3 := NextRetryAt(3, 1000)

		assert.True(t, retry3.After(retry1) || retry3.Equal(retry1))
	})
}
