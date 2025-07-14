package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock time for testing
var mockNow time.Time

// Override the nowFn for testing
func init() {
	nowFn = func() time.Time {
		return mockNow
	}
}

// Helper to advance mock time
func advanceTime(duration time.Duration) {
	mockNow = mockNow.Add(duration)
}

func TestGetLimitFirstCall(t *testing.T) {
	// Setup
	mockNow = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedLimit := 42
	mockGetter := func(ctx context.Context, key string) (int, error) { return expectedLimit, nil }
	cl := newCachedLimit(mockGetter, 10*time.Minute)

	// Execute
	limit, _, err := cl.get(context.Background(), "test-key")

	// Verify
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if limit != expectedLimit {
		t.Errorf("expected limit %d, got %d", expectedLimit, limit)
	}

	if cl.limitTimestamp != mockNow {
		t.Errorf("expected timestamp to be updated to %v, got %v", mockNow, cl.limitTimestamp)
	}
}

func TestGetLimitMultipleCalls(t *testing.T) {
	ctx := context.Background()
	// Setup
	mockNow = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	initialLimit := 42
	callCount := 0
	ttl := 10 * time.Minute

	mockGetter := func(ctx context.Context, key string) (int, error) {
		callCount++
		return initialLimit + callCount, nil
	}

	cl := newCachedLimit(mockGetter, ttl)

	// First call to populate cache
	firstLimit, changed, _ := cl.get(ctx, "test-key")
	assert.Equal(t, initialLimit+1, firstLimit, "First limit should be initialLimit+1")
	assert.True(t, changed, "First call should indicate limit changed")
	assert.Equal(t, 1, callCount, "Getter should be called once initially")

	// Make several more calls while within TTL - should use cache
	for i := 0; i < 5; i++ {
		// Advance time slightly but stay within TTL
		advanceTime(1 * time.Minute)

		cachedLimit, changed, err := cl.get(ctx, "test-key")
		require.NoError(t, err, "Should not error with cached value")
		assert.Equal(t, firstLimit, cachedLimit, "Should return cached limit")
		assert.False(t, changed, "Cached value should not indicate change")
	}

	// Verify getter was still only called once
	assert.Equal(t, 1, callCount, "Getter should still only be called once despite multiple get calls")

	// Now advance time past TTL
	advanceTime(6 * time.Minute)

	// This call should refresh the cache
	secondLimit, changed, err := cl.get(ctx, "test-key")
	require.NoError(t, err, "Should not error when refreshing")
	assert.Equal(t, initialLimit+2, secondLimit, "New limit should be initialLimit+2")
	assert.True(t, changed, "Second refresh should indicate limit changed")
	assert.Equal(t, 2, callCount, "Getter should be called a second time after TTL expires")

	// Make several more calls with the new cache
	for i := 0; i < 3; i++ {
		// Advance time slightly but stay within new TTL
		advanceTime(2 * time.Minute)

		cachedLimit, changed, err := cl.get(ctx, "test-key")
		require.NoError(t, err, "Should not error with new cached value")
		assert.Equal(t, secondLimit, cachedLimit, "Should return new cached limit")
		assert.False(t, changed, "Cached value should not indicate change")
	}

	// Verify getter was still only called twice total
	assert.Equal(t, 2, callCount, "Getter should only be called twice despite multiple additional get calls")
}

func TestGetLimitErrorThenSuccess(t *testing.T) {
	ctx := context.Background()
	// Setup
	mockNow = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedError := errors.New("limit service unavailable")
	shouldFail := true
	expectedLimit := 42

	mockGetter := func(ctx context.Context, key string) (int, error) {
		if shouldFail {
			shouldFail = false
			return 0, expectedError
		}
		return expectedLimit, nil
	}

	cl := newCachedLimit(mockGetter, 10*time.Minute)

	// First call - will fail
	_, _, firstErr := cl.get(ctx, "test-key")

	// Advance time past TTL
	advanceTime(15 * time.Minute)

	// Second call - should succeed
	limit, changed, err := cl.get(ctx, "test-key")

	// Verify
	if firstErr != expectedError {
		t.Errorf("expected first call to error with %v, got %v", expectedError, firstErr)
	}

	if err != nil {
		t.Errorf("expected second call to succeed, got error: %v", err)
	}

	if limit != expectedLimit {
		t.Errorf("expected limit %d, got %d", expectedLimit, limit)
	}

	assert.True(t, changed, "Limit should be marked as changed after successful refresh")
}
