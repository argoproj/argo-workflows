package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// executeResourceAgent routes an agent-based resource template (resource.mode: agent) onto the
// shared per-workflow resource agent pod via the WorkflowTaskSet, mirroring executeHTTPTemplate.
func (woc *wfOperationCtx) executeResourceAgent(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) *wfv1.NodeStatus {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		_, node = woc.initializeExecutableNode(ctx, nodeName, wfv1.NodeTypeResourceAgent, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending, opts.nodeFlag, true)
	}
	if !node.Fulfilled() {
		// The agent applies exactly one object per node; a multi-document inline manifest would be
		// silently partly applied (only doc 1 created, reported Succeeded). Reject it loudly here,
		// matching the agent's own single-doc guard. setOwnerReference is injected by the agent's
		// create path (withAgentMetadata) so it also covers manifestFrom, whose content we never see.
		if common.ManifestDocCount([]byte(tmpl.Resource.Manifest)) > 1 {
			return woc.markNodeError(ctx, nodeName, fmt.Errorf("agent-based resource templates support only a single manifest document"))
		}
		// The shared agent pod runs under `<workflow-sa>-resource-agent`; a template-level
		// service account cannot be honored there. Validation rejects this at submit, but
		// templates can reach here without lint (e.g. resolved workflowtemplate refs), so
		// refuse loudly rather than run the action under an account the user did not ask for.
		if tmpl.ServiceAccountName != "" || (tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "") {
			return woc.markNodeError(ctx, nodeName, fmt.Errorf("agent-based resource templates cannot use a template-level serviceAccountName or executor.serviceAccountName; the agent pod runs under the workflow service account name suffixed with -resource-agent"))
		}
		woc.taskSet[node.ID] = *tmpl
	}
	return node
}
