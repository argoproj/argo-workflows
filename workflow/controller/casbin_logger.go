package controller

import (
	casbinlog "github.com/casbin/casbin/v2/log"
	log "github.com/sirupsen/logrus"
)

type casbinLogger struct{}

func (c casbinLogger) EnableLog(b bool) {}

func (c casbinLogger) IsEnabled() bool { return true }

func (c casbinLogger) LogModel(model [][]string) {
	log.WithField("model", model).Info("Model")
}

func (c casbinLogger) LogEnforce(matcher string, request []interface{}, result bool, explains [][]string) {
	log.WithField("matcher", matcher).
		WithField("request", request).
		WithField("result", result).
		WithField("explains", explains).
		Info("Enforce")
}

func (c casbinLogger) LogRole(roles []string) {
	log.WithField("roles", roles).Info("Roles")
}

func (c casbinLogger) LogPolicy(policy map[string][][]string) {
	log.WithField("policy", policy).Info("Policy")
}

var casbinLoggerInstance casbinlog.Logger = &casbinLogger{}
