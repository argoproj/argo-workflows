package controller

type policyDef struct {
	workflowNamespace string
	cluster           string
	namespace         string
	act               act // bitmap, like chmod file perms
}

type act int

var (
	actRead  act = 0b01
	actWrite act = 0b10
)

func (p policyDef) matches(workflowNamespace, cluster, namespace string, act act) bool {
	return cluster == p.cluster &&
		(workflowNamespace == p.workflowNamespace || p.workflowNamespace == "") &&
		(namespace == p.namespace || p.namespace == "") &&
		act&p.act > 0
}
