package sync

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// workflowThrottleKey represents a key used in the throttle queue
// Format: "workflowKey/priority/creationTime/action"
type workflowThrottleKey = string

// ThrottleAction represents the action type for throttle operations
type ThrottleAction string

const (
	ThrottleActionAdd    ThrottleAction = "add"
	ThrottleActionUpdate ThrottleAction = "update"
	ThrottleActionDelete ThrottleAction = "delete"
)

// NewThrottleKey creates a throttle key with workflow key, priority, creation time and action
func NewThrottleKey(key string, priority int32, creation time.Time, action ThrottleAction) workflowThrottleKey {
	// Use RFC3339 for time format to ensure parse compatibility
	return fmt.Sprintf("%s/%d/%s/%s", key, priority, creation.Format(time.RFC3339), action)
}

// ParseThrottleKey parses a throttle key back to its components
func ParseThrottleKey(throttleKey workflowThrottleKey) (key string, priority int32, creation time.Time, action ThrottleAction) {
	parts := strings.SplitN(throttleKey, "/", 4)
	if len(parts) != 4 {
		return "", 0, time.Time{}, ""
	}
	key = parts[0]
	priority64, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return "", 0, time.Time{}, ""
	}
	priority = int32(priority64)
	creation, err = time.Parse(time.RFC3339, parts[2])
	if err != nil {
		return "", 0, time.Time{}, ""
	}
	action = ThrottleAction(parts[3])
	return key, priority, creation, action
}
