package v1alpha1

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

func ParseStringToDuration(durationString string) (time.Duration, error) {
	var duration time.Duration
	// If no units are attached, treat as seconds
	if val, err := strconv.ParseInt(durationString, 10, 64); err == nil {
		const maxWholeSeconds = int64(math.MaxInt64) / int64(time.Second)
		const minWholeSeconds = int64(math.MinInt64) / int64(time.Second)
		if val > maxWholeSeconds || val < minWholeSeconds {
			return 0, fmt.Errorf("unable to parse %s as a duration: whole seconds overflow time.Duration", durationString)
		}
		duration = time.Duration(val) * time.Second
	} else if parsed, err := time.ParseDuration(durationString); err == nil {
		duration = parsed
	} else {
		return 0, fmt.Errorf("unable to parse %s as a duration: %w", durationString, err)
	}
	return duration, nil
}
