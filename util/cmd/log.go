package cmd

import (
	"log"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/sirupsen/logrus"
)

// SetLogLevel parses and sets a logrus log level
func SetLogLevel(logLevel string) {
	level, err := logging.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	logging.SetGlobalLevel(level)
	logrusLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	logrus.SetLevel(logrusLevel)
}
