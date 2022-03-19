package controller

import "fmt"

type policyDef struct {
	workflowNamespace string
	cluster           string
	namespace         string
	role              role // bitmap, like chmod file perms
}

type role int

var (
	roleRead  role = 0b01
	roleWrite role = 0b10
)

func (p policyDef) matches(workflowNamespace, cluster, namespace string, role role) bool {
	return cluster == p.cluster &&
		(workflowNamespace == p.workflowNamespace || p.workflowNamespace == "") &&
		(namespace == p.namespace || p.namespace == "") &&
		role&p.role > 0
}

func (p policyDef) String() string {
	return fmt.Sprintf("%s,%s,%s,%d", p.workflowNamespace, p.cluster, p.namespace, p.role)
}
