package common

import (
	"log"
	"os"
)

var DebugLog *log.Logger
var InfoLog *log.Logger
var WarnLog *log.Logger
var ErrorLog *log.Logger

// Init the loggers.
func InitLoggers(prefix string) {
	// Log to stdout during development. Later switch to log to syslog.
	DebugLog = log.New(os.Stdout, "["+prefix+"-debug] ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLog = log.New(os.Stdout, "["+prefix+"-info] ", log.Ldate|log.Ltime|log.Lshortfile)
	WarnLog = log.New(os.Stdout, "["+prefix+"-warn] ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(os.Stdout, "["+prefix+"-error] ", log.Ldate|log.Ltime|log.Lshortfile)
}
