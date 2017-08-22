// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops

import (
	"applatix.io/axops/utils"
	"applatix.io/common"
	"log"
)

var DebugLog *log.Logger
var InfoLog *log.Logger
var ErrorLog *log.Logger

// Init the loggers.
func InitLoggers() {

	common.InitLoggers("AXOPS")
	// Log to stdout during development. Later switch to log to syslog.
	DebugLog = common.DebugLog
	InfoLog = common.InfoLog
	ErrorLog = common.ErrorLog

	utils.DebugLog = DebugLog
	utils.InfoLog = InfoLog
	utils.ErrorLog = ErrorLog
}
