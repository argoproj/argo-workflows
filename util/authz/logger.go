package authz

import (
	casbinlog "github.com/casbin/casbin/v2/log"
	log "github.com/sirupsen/logrus"
)

type logger struct{}

func (l logger) EnableLog(bool) {}

func (l logger) IsEnabled() bool { return true }

func (l logger) LogModel(model [][]string) {
	log.WithField("model", model).Info("Model")
}

func (l logger) LogEnforce(matcher string, request []interface{}, result bool, explains [][]string) {
	log.WithField("matcher", matcher).
		WithField("request", request).
		WithField("result", result).
		WithField("explains", explains).
		Info("Enforce")
}

func (l logger) LogRole(roles []string) {
	log.WithField("roles", roles).
		Info("Roles")
}

func (l logger) LogPolicy(policy map[string][][]string) {
	log.WithField("policy", policy).
		Info("Policy")
}

var Logger casbinlog.Logger = &logger{}
