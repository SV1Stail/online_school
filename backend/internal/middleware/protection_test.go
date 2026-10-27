package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimiterRefillsTokens(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(1, 2, 10)
	now := time.Date(2026, time.October, 25, 12, 0, 0, 0, time.UTC)
	require.True(t, limiter.allow("127.0.0.1", now))
	require.True(t, limiter.allow("127.0.0.1", now))
	require.False(t, limiter.allow("127.0.0.1", now))
	require.True(t, limiter.allow("127.0.0.1", now.Add(time.Second)))
}

func TestRateLimiterBoundsClientMap(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(1, 1, 1)
	now := time.Date(2026, time.October, 25, 12, 0, 0, 0, time.UTC)
	require.True(t, limiter.allow("first", now))
	require.False(t, limiter.allow("second", now))
}
