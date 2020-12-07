package env

import (
	"os"
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
