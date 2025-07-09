package cmd

import (
	"log"

	"github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// SetLogLevel parses and sets a logrus log level
func SetLogLevel(logLevel string) {
	level, err := logging.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	logging.SetGlobalLevel(level)
	switch logLevel {
	case "trace":
		logLevel = "debug"
	case "print":
		logLevel = "info"
	}
	logrusLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	logrus.SetLevel(logrusLevel)
}
