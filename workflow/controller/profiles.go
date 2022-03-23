package controller

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type profiles map[profileKey]*profile

func (p profiles) find(workflowNamespace, cluster, namespace string) (*profile, error) {
	for _, p := range p {
		if p.matches(workflowNamespace, cluster, namespace) {
			log.Infof("%s,%s,%s -> %s,%s,%s", workflowNamespace, cluster, namespace, p.workflowNamespace, p.cluster, p.namespace)
			return p, nil
		}
	}
	return nil, fmt.Errorf("profile not found for policy %s,%s,%s", workflowNamespace, cluster, namespace)
}

func (p profiles) local() *profile {
	return p[localProfileKey]
}
