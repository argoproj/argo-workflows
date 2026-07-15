package humanize

import (
	"fmt"
	"math"
	"strings"
	"time"

	gohumanize "github.com/dustin/go-humanize"
)

func Timestamp(ts time.Time) string {
	return fmt.Sprintf("%s (%s)", ts.Format("Mon Jan 02 15:04:05 -0700"), gohumanize.Time(ts))
}

var relativeMagnitudes = []gohumanize.RelTimeMagnitude{
	{D: time.Second, Format: "0 seconds", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1 second %s", DivBy: 1},
	{D: time.Minute, Format: "%d seconds %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1 minute %s", DivBy: 1},
	{D: time.Hour, Format: "%d minutes %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1 hour %s", DivBy: 1},
	{D: gohumanize.Day, Format: "%d hours %s", DivBy: time.Hour},
	{D: 2 * gohumanize.Day, Format: "1 day %s", DivBy: 1},
	{D: gohumanize.Week, Format: "%d days %s", DivBy: gohumanize.Day},
	{D: 2 * gohumanize.Week, Format: "1 week %s", DivBy: 1},
	{D: gohumanize.Month, Format: "%d weeks %s", DivBy: gohumanize.Week},
	{D: 2 * gohumanize.Month, Format: "1 month %s", DivBy: 1},
	{D: gohumanize.Year, Format: "%d months %s", DivBy: gohumanize.Month},
	{D: 18 * gohumanize.Month, Format: "1 year %s", DivBy: 1},
	{D: 2 * gohumanize.Year, Format: "2 years %s", DivBy: 1},
	{D: gohumanize.LongTime, Format: "%d years %s", DivBy: gohumanize.Year},
	{D: math.MaxInt64, Format: "a long while %s", DivBy: 1},
}

// TruncatedDuration returns a duration truncated to a single unit
func TruncatedDuration(d time.Duration) string {
	start := time.Time{}
	finish := start.Add(d)
	return strings.TrimSpace(gohumanize.CustomRelTime(start, finish, "", "", relativeMagnitudes))
}

// Duration humanizes time.Duration output to a meaningful value with up to two units
func Duration(d time.Duration) string {
	if d.Seconds() < 60.0 {
		return TruncatedDuration(d)
	}
	if d.Minutes() < 60.0 {
		remainingSeconds := int64(math.Mod(d.Seconds(), 60))
		return fmt.Sprintf("%s %d seconds", TruncatedDuration(d), remainingSeconds)
	}
	if d.Hours() < 24.0 {
		remainingMinutes := int64(math.Mod(d.Minutes(), 60))
		return fmt.Sprintf("%s %d minutes", TruncatedDuration(d), remainingMinutes)
	}
	remainingHours := int64(math.Mod(d.Hours(), 24))
	return fmt.Sprintf("%s %d hours", TruncatedDuration(d), remainingHours)
}

// RelativeDuration returns a formatted duration from the relative times
func RelativeDuration(start, finish time.Time) string {
	if finish.IsZero() && !start.IsZero() {
		finish = time.Now().UTC()
	}
	return Duration(finish.Sub(start))
}

var shortTimeMagnitudes = []gohumanize.RelTimeMagnitude{
	{D: time.Second, Format: "0s", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1s %s", DivBy: 1},
	{D: time.Minute, Format: "%ds %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1m %s", DivBy: 1},
	{D: time.Hour, Format: "%dm %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1h %s", DivBy: 1},
	{D: gohumanize.Day, Format: "%dh %s", DivBy: time.Hour},
	{D: 2 * gohumanize.Day, Format: "1d %s", DivBy: 1},
	{D: gohumanize.Week, Format: "%dd %s", DivBy: gohumanize.Day},
}

// RelativeDurationShort returns a relative duration in short format
func RelativeDurationShort(start, finish time.Time) string {
	if finish.IsZero() && !start.IsZero() {
		finish = time.Now().UTC()
	}
	return strings.TrimSpace(gohumanize.CustomRelTime(start, finish, "", "", shortTimeMagnitudes))
}
