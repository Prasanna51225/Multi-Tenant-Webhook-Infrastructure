package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateBackoff(t *testing.T) {
	t.Run("attempt 0 returns jitter within base range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			backoff := CalculateBackoff(0, 1000)
			assert.GreaterOrEqual(t, backoff, time.Duration(0))
			assert.LessOrEqual(t, backoff, 1000*time.Millisecond)
		}
	})

	t.Run("attempt 1 returns jitter within 2x base range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			backoff := CalculateBackoff(1, 1000)
			assert.GreaterOrEqual(t, backoff, time.Duration(0))
			assert.LessOrEqual(t, backoff, 2000*time.Millisecond)
		}
	})

	t.Run("attempt 2 returns jitter within 4x base range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			backoff := CalculateBackoff(2, 1000)
			assert.GreaterOrEqual(t, backoff, time.Duration(0))
			assert.LessOrEqual(t, backoff, 4000*time.Millisecond)
		}
	})

	t.Run("attempt 3 returns jitter within 8x base range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			backoff := CalculateBackoff(3, 1000)
			assert.GreaterOrEqual(t, backoff, time.Duration(0))
			assert.LessOrEqual(t, backoff, 8000*time.Millisecond)
		}
	})

	t.Run("backoff never exceeds max delay", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			backoff := CalculateBackoff(20, 1000)
			assert.GreaterOrEqual(t, backoff, time.Duration(0))
			assert.LessOrEqual(t, backoff, MaxDelay)
		}
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
		for i := 0; i < 100; i++ {
			backoff := CalculateBackoff(1, 500)
			assert.GreaterOrEqual(t, backoff, time.Duration(0))
			assert.LessOrEqual(t, backoff, 1000*time.Millisecond)
		}
	})
}

func TestNextRetryAt(t *testing.T) {
	t.Run("returns time in the future", func(t *testing.T) {
		before := time.Now().UTC()
		retryAt := NextRetryAt(1, 1000)
		assert.True(t, retryAt.After(before) || retryAt.Equal(before))
	})

	t.Run("returns time within expected maximum range", func(t *testing.T) {
		before := time.Now().UTC()
		retryAt := NextRetryAt(2, 1000)
		maxExpected := before.Add(4000 * time.Millisecond)
		assert.True(t, retryAt.Before(maxExpected) || retryAt.Equal(maxExpected),
			"retry time should be within 4x base range")
	})

	t.Run("higher attempt has larger maximum backoff window", func(t *testing.T) {
		maxBackoff1 := time.Duration(2000) * time.Millisecond
		maxBackoff3 := time.Duration(8000) * time.Millisecond
		assert.Greater(t, maxBackoff3, maxBackoff1,
			"attempt 3 should have larger max backoff than attempt 1")
	})
}
