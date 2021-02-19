package plugin

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type TemplateExecutor interface {
	Init(req InitReq, resp *InitResp) error
	/*
		The function is called when an plugin template needs executing.
		It is only called if the plugin has registered its ability to execute the template
		in InitResp.Templates.

		Implementations must be able to complete all requests rapidly, as the controller allocates 30s per workflow.
		A workflow with 1000 nodes, that takes 1s per node, will never be able to finish.

		Typically implementations will offload work to an async process.

		This will be invoked one per reconciliation, so implementation must be idempotent.

		Options:
		 * return an error
		 * set the phase of the node status
		 * do nothing
	*/
	ExecuteNode(req ExecuteNodeReq, resp *wfv1.NodeStatus) error
}

type InitReq struct{}

type InitResp struct {
	PluginTemplateTypes []string `json:"pluginTemplateTypes"`
}

type ExecuteNodeReq struct {
	Workflow metav1.ObjectMeta `json:"workflow"`
	Template wfv1.Template     `json:"template"`
	Node     wfv1.NodeStatus   `json:"node"`
}
