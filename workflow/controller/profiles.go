package controller

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
)

const localProfileKey cache.ExplicitKey = ""

type profiles map[cache.ExplicitKey]*profile

func (p profiles) find(workflowNamespace, cluster, namespace string, act act) (*profile, error) {
	for _, p := range p {
		if p.matches(workflowNamespace, cluster, namespace, act) {
			log.Infof("%s,%s,%s,%v -> %s,%s,%s,%v", workflowNamespace, cluster, namespace, act, p.workflowNamespace, p.cluster, p.namespace, p.act)
			return p, nil
		}
	}
	return nil, fmt.Errorf("profile not found for %s,%s,%s,%v", workflowNamespace, cluster, namespace, act)
}

func (p profiles) local() *profile {
	return p[localProfileKey]
}
