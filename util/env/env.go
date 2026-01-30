package env

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

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
