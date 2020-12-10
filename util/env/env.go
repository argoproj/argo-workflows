package env

import (
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func LookupEnvDurationOr(key string, o time.Duration) time.Duration {
	v, found := os.LookupEnv(key)
	if found {
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
	if found {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.WithField(key, v).WithError(err).Panic("failed to parse")
		} else {
			return i
		}
	}
	return o
}
