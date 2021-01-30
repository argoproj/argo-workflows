package retry

import (
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"

	envutil "github.com/argoproj/argo/v2/util/env"
)

// BackoffSettings obtains the retry backoff settings when retrying API calls.
func BackoffSettings() wait.Backoff {
	// Below are the default retry backoff settings when retrying API calls
	// Retry   Seconds
	//     1      0.01
	//     2      0.03
	//     3      0.07
	//     4      0.15
	//     5      0.31
	steps := 5
	defaultDuration := 10 * time.Millisecond
	factor := 2.

	stepsStr, found := os.LookupEnv("RETRY_BACKOFF_STEPS")
	if found {
		convertedSteps, err := strconv.Atoi(stepsStr)
		if err != nil {
			log.WithField("RETRY_BACKOFF_STEPS", stepsStr).WithError(err).Warn("failed to convert to int")
		} else {
			steps = convertedSteps
		}
	}

	duration := envutil.LookupEnvDurationOr("RETRY_BACKOFF_DURATION", defaultDuration)

	factorStr, found := os.LookupEnv("RETRY_BACKOFF_FACTOR")
	if found {
		convertedFactor, err := strconv.ParseFloat(factorStr, 64)
		if err != nil {
			log.WithField("RETRY_BACKOFF_FACTOR", factorStr).WithError(err).Warn("failed to convert to float64")
		} else {
			factor = convertedFactor
		}
	}

	return wait.Backoff{
		Steps:    steps,
		Duration: duration,
		Factor:   factor,
	}
}
