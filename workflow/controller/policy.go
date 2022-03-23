package controller

import "fmt"

type policy struct {
	workflowNamespace string
	cluster           string
	namespace         string
}

func (p policy) matches(workflowNamespace, cluster, namespace string) bool {
	return cluster == p.cluster &&
		(workflowNamespace == p.workflowNamespace || p.workflowNamespace == "") &&
		(namespace == p.namespace || p.namespace == "")
}

func (p policy) String() string {
	return fmt.Sprintf("%s,%s,%s", p.workflowNamespace, p.cluster, p.namespace)
}
