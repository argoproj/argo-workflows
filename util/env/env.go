package env

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// LookupEnvPositiveMilliseconds returns a positive integer millisecond value as a
// duration, or nil for an unset or empty variable. Invalid or overflowing values
// return an error rather than panicking.
func LookupEnvPositiveMilliseconds(key string) (*time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, nil
	}
	// Bound milliseconds before conversion to avoid overflowing time.Duration.
	const maxMilliseconds = math.MaxInt64 / int64(time.Millisecond)
	milliseconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || milliseconds <= 0 || milliseconds > maxMilliseconds {
		return nil, fmt.Errorf("%s must be an integer between 1 and %d; got %q", key, maxMilliseconds, value)
	}
	delay := time.Duration(milliseconds) * time.Millisecond
	return &delay, nil
}

func LookupEnvDurationOr(ctx context.Context, key string, o time.Duration) time.Duration {
	logger := logging.RequireLoggerFromContext(ctx)
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
		logger = logger.WithField(key, v).WithError(err)
		logger.WithPanic().Error(ctx, "failed to parse")
	}
	return o
}

func LookupEnvIntOr(ctx context.Context, key string, o int) int {
	logger := logging.RequireLoggerFromContext(ctx)
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := strconv.Atoi(v)
		if err == nil {
			return d
		}
		logger = logger.WithField(key, v).WithError(err)
		logger.WithPanic().Error(ctx, "failed to convert to int")
	}
	return o
}

func LookupEnvFloatOr(ctx context.Context, key string, o float64) float64 {
	logger := logging.RequireLoggerFromContext(ctx)
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return d
		}
		logger = logger.WithField(key, v).WithError(err)
		logger.WithPanic().Error(ctx, "failed to convert to float")
	}
	return o
}

func LookupEnvStringOr(key string, o string) string {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		return v
	}
	return o
}
