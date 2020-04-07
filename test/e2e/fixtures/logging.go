package fixtures

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

/*
The intention of this file is to write to stdout at INFO level and a file at DEBUG level.

This is no straight forward, as by default all logger must have the same log level.
*/
func init() {
	log.SetOutput(ioutil.Discard)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.AddHook(stdout)
	log.AddHook(file)
}
