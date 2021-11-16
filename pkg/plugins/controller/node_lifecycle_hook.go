package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// swagger:parameters nodePreExecute
type NodePreExecuteRequest struct {
	// in: body
	// Required: true
	Body NodePreExecuteArgs
}

type NodePreExecuteArgs struct {
	// Required: true
	Workflow *Workflow `json:"workflow"`
	// Required: true
	Template *wfv1.Template `json:"template"`
	// Required: true
	Node *wfv1.NodeStatus `json:"node"`
}

// swagger:response nodePreExecute
type NodePreExecuteResponse struct {
	// in: body
	Body NodePreExecuteReply
}

type NodePreExecuteReply struct {
	Node *wfv1.NodeResult `json:"node,omitempty"`
}

// swagger:parameters nodePostExecute
type NodePostExecuteRequest struct {
	// in: body
	// Required: true
	Body NodePostExecuteArgs
}

type NodePostExecuteArgs struct {
	Workflow *Workflow        `json:"workflow"`
	Template *wfv1.Template   `json:"template"`
	Old      *wfv1.NodeStatus `json:"old"`
	New      *wfv1.NodeStatus `json:"new"`
}

// swagger:response nodePostExecute
type NodePostExecuteResponse struct {
	// in: body
	Body NodePostExecuteReply
}

type NodePostExecuteReply struct {
}

type NodeLifecycleHook interface {
	// NodePreExecute is called when executing a template. It will called multiple times.
	// swagger:route POST /node.preExecute nodePreExecute
	//     Responses:
	//       200: nodePreExecute
	NodePreExecute(args NodePreExecuteArgs, reply *NodePreExecuteReply) error

	// NodePostExecute is called after executing a template.
	// swagger:route POST /node.postExecute nodePostExecute
	//     Responses:
	//       200: nodePostExecute
	NodePostExecute(args NodePostExecuteArgs, reply *NodePostExecuteReply) error
}
