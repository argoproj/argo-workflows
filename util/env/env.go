package env

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func LookupEnvDurationOr(ctx context.Context, key string, o time.Duration) time.Duration {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			logger := logging.GetLoggerFromContext(ctx)
			logger = logger.WithField(ctx, key, v).WithError(ctx, err)
			logger.Panic(ctx, "failed to parse")
		} else {
			return d
		}
	}
	return o
}

func LookupEnvIntOr(ctx context.Context, key string, o int) int {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := strconv.Atoi(v)
		if err != nil {
			logger := logging.GetLoggerFromContext(ctx)
			logger = logger.WithField(ctx, key, v).WithError(ctx, err)
			logger.Panic(ctx, "failed to convert to int")
		} else {
			return d
		}
	}
	return o
}

func LookupEnvFloatOr(ctx context.Context, key string, o float64) float64 {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := strconv.ParseFloat(v, 64)
		if err != nil {
			logger := logging.GetLoggerFromContext(ctx)
			logger = logger.WithField(ctx, key, v).WithError(ctx, err)
			logger.Panic(ctx, "failed to convert to float")
		} else {
			return d
		}
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
