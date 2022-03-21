package controller

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type profiles map[profileKey]*profile

func (p profiles) find(workflowNamespace, cluster, namespace string, act role) (*profile, error) {
	for _, p := range p {
		if p.matches(workflowNamespace, cluster, namespace, act) {
			log.Infof("%s,%s,%s,%v -> %s,%s,%s,%v", workflowNamespace, cluster, namespace, act, p.workflowNamespace, p.cluster, p.namespace, p.role)
			return p, nil
		}
	}
	return nil, fmt.Errorf("profile not found for policy %s,%s,%s,%v", workflowNamespace, cluster, namespace, act)
}

func (p profiles) local() *profile {
	return p[localProfileKey]
}
