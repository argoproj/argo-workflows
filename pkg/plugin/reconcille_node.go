package plugin

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
)

/*
This functions is called whenever a reconciliation happens for a plugin template node.

It will be call once per reconciliation until it returns that the nodes has completed.

Options:
 * return an error
 * set the phase of the node status
 * do nothing
*/
type ReconcileNodeFunc = func(req ReconcileNodeReq, resp *wfv1.NodeStatus) error

type ReconcileNodeReq struct {
	Workflow metav1.ObjectMeta `json:"workflow"`
	Template wfv1.Template     `json:"template"`
	Node     wfv1.NodeStatus   `json:"node"`
}
