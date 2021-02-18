package e2e

import log "github.com/sirupsen/logrus"

type httpLogger struct{}

func (d *httpLogger) Logf(fmt string, args ...interface{}) {
	log.Debugf(fmt, args...)
}
