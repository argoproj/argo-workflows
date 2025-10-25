package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewThrottleKey(t *testing.T) {
	key := "test-namespace/test-workflow"
	priority := int32(5)
	creation := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	action := ThrottleActionAdd

	throttleKey := NewThrottleKey(key, priority, creation, action)
	expected := "test-namespace/test-workflow/5/2023-01-01T12:00:00Z/add"
	assert.Equal(t, expected, throttleKey, "Throttle key should match expected format")
}

func TestParseThrottleKey(t *testing.T) {
	throttleKey := "test-namespace/test-workflow/5/2023-01-01T12:00:00Z/add"
	key, priority, creation, action := ParseThrottleKey(throttleKey)

	expectedKey := "test-namespace/test-workflow"
	expectedPriority := int32(5)
	expectedCreation := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	expectedAction := ThrottleActionAdd

	assert.Equal(t, expectedKey, key, "Parsed key should match expected")
	assert.Equal(t, expectedPriority, priority, "Parsed priority should match expected")
	assert.Equal(t, expectedCreation, creation, "Parsed creation time should match expected")
	assert.Equal(t, expectedAction, action, "Parsed action should match expected")
}

func TestParseThrottleKeyInvalid(t *testing.T) {
	tests := []struct {
		name        string
		throttleKey string
	}{
		{"empty", ""},
		{"too few parts", "test-namespace/test-workflow"},
		{"too many parts", "test-namespace/test-workflow/5/2023-01-01T12:00:00Z/add/extra"},
		{"invalid priority", "test-namespace/test-workflow/invalid/2023-01-01T12:00:00Z/add"},
		{"invalid time", "test-namespace/test-workflow/5/invalid-time/add"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, priority, creation, action := ParseThrottleKey(tt.throttleKey)
			assert.Empty(t, key, "Key should be empty for invalid throttle key")
			assert.Equal(t, int32(0), priority, "Priority should be 0 for invalid throttle key")
			assert.True(t, creation.IsZero(), "Creation time should be zero for invalid throttle key")
			assert.Empty(t, action, "Action should be empty for invalid throttle key")
		})
	}
}

func TestThrottleKeyRoundTrip(t *testing.T) {
	originalKey := "test-namespace/test-workflow"
	originalPriority := int32(10)
	originalCreation := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	originalAction := ThrottleActionUpdate

	throttleKey := NewThrottleKey(originalKey, originalPriority, originalCreation, originalAction)
	parsedKey, parsedPriority, parsedCreation, parsedAction := ParseThrottleKey(throttleKey)

	assert.Equal(t, originalKey, parsedKey, "Round-trip key should match original")
	assert.Equal(t, originalPriority, parsedPriority, "Round-trip priority should match original")
	assert.True(t, originalCreation.Equal(parsedCreation), "Round-trip creation time should match original")
	assert.Equal(t, originalAction, parsedAction, "Round-trip action should match original")
}
