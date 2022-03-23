package controller

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type profiles map[profileKey]*profile

func (ps profiles) find(workflowNamespace, cluster, namespace string) (*profile, error) {
	for _, p := range ps {
		if p.matches(workflowNamespace, cluster, namespace) {
			log.Infof("%s,%s,%s -> %s,%s,%s", workflowNamespace, cluster, namespace, p.workflowNamespace, p.cluster, p.namespace)
			return p, nil
		}
	}
	return nil, fmt.Errorf("profile not found for policy %s,%s,%s", workflowNamespace, cluster, namespace)
}

func (ps profiles) run(done <-chan struct{}) {
	for _, p := range ps {
		p.run(done)
	}
}

func (ps profiles) hasSynced() bool {
	for _, p := range ps {
		if !p.hasSynced() {
			return false
		}
	}
	return true
}
