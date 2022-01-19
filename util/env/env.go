package env

import (
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func LookupEnvDurationOr(key string, o time.Duration) time.Duration {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.WithField(key, v).WithError(err).Panic("failed to parse")
		} else {
			return d
		}
	}
	return o
}

func LookupEnvIntOr(key string, o int) int {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := strconv.Atoi(v)
		if err != nil {
			log.WithField(key, v).WithError(err).Panic("failed to convert to int")
		} else {
			return d
		}
	}
	return o
}

func LookupEnvFloatOr(key string, o float64) float64 {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		d, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.WithField(key, v).WithError(err).Panic("failed to convert to float")
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
