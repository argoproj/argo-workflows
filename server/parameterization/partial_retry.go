package parameterization

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	native "github.com/argoproj/argo-workflows/v4/workflow/util"
)

// PartialRetryPlan is a checked native retry, not a second scheduling plan.
// The receipt freezes the original execution before native retry removes nodes.
type PartialRetryPlan = parameterapi.PartialRetryPlan

type PartialRetryRequest struct {
	Workflow string `json:"workflow"`
	UID      string `json:"uid"`
	Revision string `json:"revision"`
}

func (h Handler) partialRetryPreview(ctx context.Context, s Store, req PartialRetryRequest) (*PartialRetryPlan, *wfv1.Workflow, error) {
	plan, wf, _, err := h.partialRetryResolved(ctx, s, req)
	return plan, wf, err
}

func (h Handler) partialRetryResolved(ctx context.Context, s Store, req PartialRetryRequest) (*PartialRetryPlan, *wfv1.Workflow, *Preview, error) {
	wf, err := h.Workflows.GetWorkflow(ctx, &workflowapi.WorkflowGetRequest{Namespace: s.Namespace, Name: req.Workflow})
	if err != nil {
		return nil, nil, nil, err
	}
	if string(wf.UID) != req.UID {
		return nil, nil, nil, fmt.Errorf("workflow identity changed")
	}
	source, err := h.editContext(ctx, s, SourceExecution{Workflow: req.Workflow, UID: req.UID})
	if err != nil {
		return nil, nil, nil, err
	}
	next, err := s.get(ctx, "parameterrevisions", req.Revision)
	if err != nil {
		return nil, nil, nil, err
	}
	var rev Revision
	if err = decode(next, &rev); err != nil {
		return nil, nil, nil, err
	}
	if rev.Set != source.Set.GetName() {
		return nil, nil, nil, fmt.Errorf("retry revision belongs to another parameter set")
	}
	preview, err := s.Preview(ctx, source.Binding, req.Revision)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(preview.Errors) > 0 {
		return nil, nil, nil, fmt.Errorf("retry revision is invalid")
	}
	plan, err := planPartialRetry(ctx, wf, preview.Workflow)
	if err != nil {
		return nil, nil, nil, err
	}
	if err = s.verifyRetainedArtifacts(ctx, wf, plan); err != nil {
		return nil, nil, nil, err
	}
	plan.Revision = req.Revision
	plan.PreviousRevision = wf.Annotations[p.Group+"/revision"]
	plan.Binding = source.Binding
	plan.ExecutionDigest = preview.ExecutionDigest
	plan.Digest = p.Digest(plan)
	return plan, wf, preview, nil
}

func planPartialRetry(ctx context.Context, wf, next *wfv1.Workflow) (*PartialRetryPlan, error) {
	if wf.Status.Phase != wfv1.WorkflowFailed && wf.Status.Phase != wfv1.WorkflowError && wf.Status.Phase != wfv1.WorkflowSucceeded {
		return nil, fmt.Errorf("partial retry requires a completed workflow")
	}
	if wf.Status.CompressedNodes != "" || wf.Status.OffloadNodeStatusVersion != "" {
		return nil, fmt.Errorf("retry requires hydrated execution records")
	}
	if len(wf.Status.Nodes) == 0 || len(wf.Status.Nodes) > 1000 {
		return nil, fmt.Errorf("partial retry needs 1-1000 observed nodes")
	}
	a, b := wf.Spec.DeepCopy(), next.Spec.DeepCopy()
	a.Arguments = wfv1.Arguments{}
	b.Arguments = wfv1.Arguments{}
	if !reflect.DeepEqual(a, b) {
		return nil, fmt.Errorf("partial retry does not support changes to frozen templates, bindings, secret versions or topology")
	}
	if a.OnExit != "" || len(a.Hooks) > 0 || len(a.VolumeClaimTemplates) > 0 || a.WorkflowTemplateRef != nil || wf.Status.StoredWorkflowSpec != nil {
		return nil, fmt.Errorf("partial retry does not support exit hooks, generated volumes or unresolved template references")
	}
	values := map[string]string{}
	for _, v := range next.Spec.Arguments.Parameters {
		if v.Value == nil {
			return nil, fmt.Errorf("retry requires resolved arguments")
		}
		values[v.Name] = v.Value.String()
	}
	old := map[string]string{}
	for _, v := range wf.Spec.Arguments.Parameters {
		if v.Value == nil {
			return nil, fmt.Errorf("original argument has no fingerprint")
		}
		old[v.Name] = v.Value.String()
	}
	if len(values) != len(old) {
		return nil, fmt.Errorf("parameter identity changed")
	}
	plan := &PartialRetryPlan{Workflow: wf.Name, UID: string(wf.UID), ResourceVersion: wf.ResourceVersion, RetainedNodes: map[string]string{}}
	for key, value := range values {
		previous, ok := old[key]
		if !ok {
			return nil, fmt.Errorf("parameter identity changed")
		}
		if value != previous {
			plan.ChangedParameters = append(plan.ChangedParameters, key)
			plan.Parameters = append(plan.Parameters, key+"="+value)
		}
	}
	sort.Strings(plan.ChangedParameters)
	sort.Strings(plan.Parameters)
	if len(plan.ChangedParameters) == 0 {
		return nil, fmt.Errorf("no execution inputs changed; use original-parameter retry")
	}
	graph, err := newRetryGraph(a, plan.ChangedParameters)
	if err != nil {
		return nil, err
	}
	plan.AffectedTasks = graph.affectedLeaves()
	if len(plan.AffectedTasks) == 0 {
		return nil, fmt.Errorf("changed input has no supported native task consumer")
	}
	selected, retained, err := graph.matchNodes(wf)
	if err != nil {
		return nil, err
	}
	plan.RetainedNodes = retained
	if len(selected) == 0 {
		return nil, fmt.Errorf("no observed affected invocation; unsupported retry target")
	}
	// Native field selectors support conjunctions. Excluding every other known
	// ID expresses the checked set without introducing a custom retry scheduler.
	excluded := []string{}
	for id := range wf.Status.Nodes {
		if !selected[id] {
			excluded = append(excluded, "id!="+id)
		}
	}
	sort.Strings(excluded)
	plan.Selector = strings.Join(excluded, ",")
	if plan.Selector == "" {
		return nil, fmt.Errorf("missing retained DAG boundary")
	}
	proof := p.NativeRetryProof{UID: string(wf.UID), ResourceVersion: wf.ResourceVersion, RetainedNodes: plan.RetainedNodes}
	after, _, err := native.FormulateRetryWorkflow(p.WithRetryProof(ctx, proof), wf, true, plan.Selector, plan.Parameters)
	if err != nil {
		return nil, err
	}
	for id, digest := range plan.RetainedNodes {
		node, exists := after.Status.Nodes[id]
		if !exists || p.ResultFingerprint(node) != digest {
			return nil, fmt.Errorf("native retry would invalidate unrelated node %s", id)
		}
	}
	return plan, nil
}

func (h Handler) retryChangedParameters(ctx context.Context, s Store, req RetryRequest) (*wfv1.Workflow, error) {
	plan, wf, preview, err := h.partialRetryResolved(ctx, s, PartialRetryRequest{Workflow: req.Workflow, UID: req.UID, Revision: req.Revision})
	if err != nil {
		return nil, err
	}
	if req.PlanDigest != plan.Digest || req.ResourceVersion != plan.ResourceVersion {
		return nil, fmt.Errorf("retry preview is stale; preview the current execution before retrying")
	}
	name := "retry-" + p.Digest([]string{req.UID, req.IdempotencyKey})[:40]
	// Typed clients omit TypeMeta after decoding. Embedded Kubernetes objects
	// must retain their GVK for structural admission and later reconstruction.
	wf = wf.DeepCopy()
	wf.APIVersion = "argoproj.io/v1alpha1"
	wf.Kind = "Workflow"
	receipt := object("ParameterRetry", name, s.Namespace, map[string]any{"requestDigest": p.Digest(req), "workflow": req.Workflow, "workflowUID": req.UID, "resourceVersion": req.ResourceVersion, "revision": req.Revision, "previousRevision": plan.PreviousRevision, "actor": s.Actor, "plan": plan, "previousWorkflow": wf, "definitions": preview.Definitions})
	if _, err = s.resource("parameterretries").Create(ctx, receipt, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
		existing, readErr := s.get(ctx, "parameterretries", name)
		if readErr != nil {
			return nil, readErr
		}
		if p.Digest(existing.Object["spec"]) != p.Digest(receipt.Object["spec"]) {
			return nil, fmt.Errorf("retry receipt identity collision")
		}
	}
	proof := p.NativeRetryProof{UID: req.UID, ResourceVersion: req.ResourceVersion, Receipt: name, Revision: req.Revision, RetainedNodes: plan.RetainedNodes}
	return h.Workflows.RetryWorkflow(p.WithRetryProof(ctx, proof), &workflowapi.WorkflowRetryRequest{Namespace: s.Namespace, Name: req.Workflow, RestartSuccessful: true, NodeFieldSelector: plan.Selector, Parameters: plan.Parameters})
}
