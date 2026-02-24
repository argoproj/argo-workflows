package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// SetLogrusLevel sets the logrus log level to match the given logging.Level.
// This is needed because argoproj/pkg uses logrus internally (e.g. stats package).
func SetLogrusLevel(level logging.Level) {
	switch level {
	case logging.Debug:
		log.SetLevel(log.DebugLevel)
	case logging.Info:
		log.SetLevel(log.InfoLevel)
	case logging.Warn:
		log.SetLevel(log.WarnLevel)
	case logging.Error:
		log.SetLevel(log.ErrorLevel)
	}
}

// SetLogrusFormatter sets the logrus formatter to match the given logging.LogType.
// This is needed because argoproj/pkg uses logrus internally (e.g. stats package).
func SetLogrusFormatter(logType logging.LogType) {
	timestampFormat := "2006-01-02T15:04:05.000Z"
	switch logType {
	case logging.JSON:
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: timestampFormat})
	default:
		log.SetFormatter(&log.TextFormatter{TimestampFormat: timestampFormat, FullTimestamp: true})
	}
}
