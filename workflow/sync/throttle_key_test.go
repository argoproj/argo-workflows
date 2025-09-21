package sync

import (
	"testing"
	"time"
)

func TestNewThrottleKey(t *testing.T) {
	key := "test-namespace/test-workflow"
	priority := int32(5)
	creation := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	action := ThrottleActionAdd

	throttleKey := NewThrottleKey(key, priority, creation, action)
	expected := "test-namespace/test-workflow/5/2023-01-01T12:00:00Z/add"
	if throttleKey != expected {
		t.Errorf("Expected %s, got %s", expected, throttleKey)
	}
}

func TestParseThrottleKey(t *testing.T) {
	throttleKey := "test-namespace/test-workflow/5/2023-01-01T12:00:00Z/add"
	key, priority, creation, action := ParseThrottleKey(throttleKey)

	expectedKey := "test-namespace/test-workflow"
	expectedPriority := int32(5)
	expectedCreation := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	expectedAction := ThrottleActionAdd

	if key != expectedKey {
		t.Errorf("Expected key %s, got %s", expectedKey, key)
	}
	if priority != expectedPriority {
		t.Errorf("Expected priority %d, got %d", expectedPriority, priority)
	}
	if !creation.Equal(expectedCreation) {
		t.Errorf("Expected creation %v, got %v", expectedCreation, creation)
	}
	if action != expectedAction {
		t.Errorf("Expected action %s, got %s", expectedAction, action)
	}
}

func TestParseThrottleKeyInvalid(t *testing.T) {
	testCases := []struct {
		name        string
		throttleKey string
	}{
		{"empty", ""},
		{"too few parts", "test-namespace/test-workflow"},
		{"too many parts", "test-namespace/test-workflow/5/2023-01-01T12:00:00Z/add/extra"},
		{"invalid priority", "test-namespace/test-workflow/invalid/2023-01-01T12:00:00Z/add"},
		{"invalid time", "test-namespace/test-workflow/5/invalid-time/add"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key, priority, creation, action := ParseThrottleKey(tc.throttleKey)
			if key != "" || priority != 0 || !creation.IsZero() || action != "" {
				t.Errorf("Expected empty values for invalid throttle key, got key=%s, priority=%d, creation=%v, action=%s",
					key, priority, creation, action)
			}
		})
	}
}

func TestThrottleKeyRoundTrip(t *testing.T) {
	originalKey := "test-namespace/test-workflow"
	originalPriority := int32(10)
	originalCreation := time.Now().UTC()
	originalAction := ThrottleActionUpdate

	throttleKey := NewThrottleKey(originalKey, originalPriority, originalCreation, originalAction)
	parsedKey, parsedPriority, parsedCreation, parsedAction := ParseThrottleKey(throttleKey)

	if parsedKey != originalKey {
		t.Errorf("Key mismatch: expected %s, got %s", originalKey, parsedKey)
	}
	if parsedPriority != originalPriority {
		t.Errorf("Priority mismatch: expected %d, got %d", originalPriority, parsedPriority)
	}
	if !parsedCreation.Equal(originalCreation) {
		t.Errorf("Creation time mismatch: expected %v, got %v", originalCreation, parsedCreation)
	}
	if parsedAction != originalAction {
		t.Errorf("Action mismatch: expected %s, got %s", originalAction, parsedAction)
	}
}
