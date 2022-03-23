package controller

import "fmt"

type policyDef struct {
	workflowNamespace string
	cluster           string
	namespace         string
}

func (p policyDef) matches(workflowNamespace, cluster, namespace string) bool {
	return cluster == p.cluster &&
		(workflowNamespace == p.workflowNamespace || p.workflowNamespace == "") &&
		(namespace == p.namespace || p.namespace == "")
}

func (p policyDef) String() string {
	return fmt.Sprintf("%s,%s,%s", p.workflowNamespace, p.cluster, p.namespace)
}
