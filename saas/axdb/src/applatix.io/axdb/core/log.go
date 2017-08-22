package core

import (
	"applatix.io/common"
	"log"
)

var debugLog *log.Logger
var infoLog *log.Logger
var warningLog *log.Logger
var errorLog *log.Logger

// Init the loggers.
func InitLoggers() {

	common.InitLoggers("AXDB")

	// Log to stdout during development. Later switch to log to syslog.
	debugLog = common.DebugLog
	infoLog = common.InfoLog
	warningLog = common.WarnLog
	errorLog = common.ErrorLog
}

func GetDebugLogger() *log.Logger {
	return debugLog
}
