package logs

import "github.com/sirupsen/logrus"

const domainKey = "domain"

var logger = logrus.StandardLogger()

func GetLogger(domain string) *logrus.Entry {
	return logger.WithField(domainKey, domain)
}
