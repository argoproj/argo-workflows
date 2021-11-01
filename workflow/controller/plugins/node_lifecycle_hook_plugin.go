package plugins

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type NodePreExecuteArgs struct {
	Workflow *wfv1.Workflow
	Template *wfv1.Template
	Node     *wfv1.NodeStatus
}

type NodePreExecuteReply struct {
	Node *wfv1.NodeStatus
}

type NodePostExecuteArgs struct {
	Workflow *wfv1.Workflow
	Template *wfv1.Template
	Node     *wfv1.NodeStatus
}

type NodePostExecuteReply struct {
}

type NodeLifecycleHook interface {
	// NodePreExecute is called when executing a template. It will called multiple times.
	// If the returned status is fulfilled, then the controller will short-circuit execution.
	NodePreExecute(args NodePreExecuteArgs, reply *NodePreExecuteReply) error
	// NodePostExecute is called after executing a template.
	NodePostExecute(args NodePostExecuteArgs, reply *NodePostExecuteReply) error
}
