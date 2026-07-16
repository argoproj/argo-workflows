package controller

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
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
		// The agent applies exactly one object per node. A multi-document inline manifest can't be
		// supported, and the setOwnerReference round-trip below (yaml.Unmarshal into a single
		// unstructured) would silently drop all but the first document — reporting Succeeded having
		// applied only doc 1. Reject it loudly here, matching the agent's own single-doc guard.
		if common.ManifestDocCount([]byte(tmpl.Resource.Manifest)) > 1 {
			return woc.markNodeError(ctx, nodeName, fmt.Errorf("agent-based resource templates support only a single manifest document"))
		}
		tmpl = tmpl.DeepCopy()
		// Mirror executeResource: inject the workflow ownerReference so the created object is
		// garbage-collected with the workflow. The agent's create path does not do this, so it
		// must happen here on the inline manifest before it goes into the taskset.
		if tmpl.Resource.SetOwnerReference && tmpl.Resource.Manifest != "" {
			obj := unstructured.Unstructured{}
			if unmarshalErr := yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &obj); unmarshalErr != nil {
				return woc.markNodeError(ctx, nodeName, unmarshalErr)
			}
			obj.SetOwnerReferences(append(obj.GetOwnerReferences(), *metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind))))
			bytes, marshalErr := yaml.Marshal(obj.Object)
			if marshalErr != nil {
				return woc.markNodeError(ctx, nodeName, marshalErr)
			}
			tmpl.Resource.Manifest = string(bytes)
		}
		woc.taskSet[node.ID] = *tmpl
	}
	return node
}
