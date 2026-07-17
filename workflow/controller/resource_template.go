package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// executeResourceMonitor routes an agent-based resource template (resource.agent: true) onto the
// shared per-workflow resource agent pod via the WorkflowTaskSet, mirroring executeHTTPTemplate.
func (woc *wfOperationCtx) executeResourceMonitor(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) *wfv1.NodeStatus {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		_, node = woc.initializeExecutableNode(ctx, nodeName, wfv1.NodeTypeResourceMonitor, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending, opts.nodeFlag, true)
	}
	if !node.Fulfilled() {
		// The agent applies exactly one object per node; a multi-document inline manifest would be
		// silently partly applied (only doc 1 created, reported Succeeded). Reject it loudly here,
		// matching the agent's own single-doc guard. setOwnerReference is injected by the agent's
		// create path (withAgentMetadata) so it also covers manifestFrom, whose content we never see.
		if common.ManifestDocCount([]byte(tmpl.Resource.Manifest)) > 1 {
			return woc.markNodeError(ctx, nodeName, fmt.Errorf("agent-based resource templates support only a single manifest document"))
		}
		woc.taskSet[node.ID] = *tmpl
	}
	return node
}
