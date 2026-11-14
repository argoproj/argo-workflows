package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// AdoptsParameterization deliberately excludes unrelated native Workflows. A
// binding in the catalog alone does not force every submission to use it.
func AdoptsParameterization(wf *wfv1.Workflow) bool {
	if wf == nil {
		return false
	}
	for _, key := range []string{"submission", "revision", "execution-digest"} {
		if _, ok := wf.Annotations[p.Group+"/"+key]; ok {
			return true
		}
	}
	return false
}

// contractSpec excludes native execution controls, not execution inputs. Status
// updates, suspend/resume, stop and termination never reread mutable definitions.
func contractSpec(wf *wfv1.Workflow) *wfv1.WorkflowSpec {
	spec := wf.Spec.DeepCopy()
	spec.Suspend = nil
	spec.Shutdown = ""
	return spec
}

func sameParameterIdentity(a, b *wfv1.Workflow) bool {
	for _, key := range []string{"submission", "revision", "execution-digest"} {
		if a.Annotations[p.Group+"/"+key] != b.Annotations[p.Group+"/"+key] {
			return false
		}
	}
	return true
}

// ValidateNativeWorkflow is shared by the API and optional Kubernetes validating
// webhook. It never creates receipts, resolves a DAG or writes Workflow status.
// CREATE requires a frozen submission produced through preview/submit. UPDATE
// retains that contract; an input change requires a checked ParameterRetry.
func (s Store) ValidateNativeWorkflow(ctx context.Context, next, old *wfv1.Workflow) error {
	if !AdoptsParameterization(next) && !AdoptsParameterization(old) {
		return nil
	}
	if next == nil || next.Namespace != s.Namespace {
		return fmt.Errorf("parameterized workflow namespace mismatch")
	}
	if old != nil {
		if next.Name != old.Name || next.UID != old.UID {
			return fmt.Errorf("parameterized workflow identity changed")
		}
		if sameParameterIdentity(next, old) && p.Digest(contractSpec(next)) == p.Digest(contractSpec(old)) {
			if resetsNativeExecution(next, old) {
				if next.Annotations[p.Group+"/retry"] != old.Annotations[p.Group+"/retry"] {
					obj, err := s.get(ctx, "parameterretries", next.Annotations[p.Group+"/retry"])
					if err != nil {
						return err
					}
					var receipt parameterapi.ParameterRetrySpec
					if err = decode(obj, &receipt); err != nil {
						return err
					}
					if receipt.Plan != nil {
						return s.validateParameterRetry(ctx, next, old)
					}
				}
				return s.validateWorkflowSecrets(ctx, old)
			}
			return nil
		}
		return s.validateParameterRetry(ctx, next, old)
	}
	name := next.Annotations[p.Group+"/submission"]
	if name == "" {
		return fmt.Errorf("parameterized workflow requires a frozen WorkflowSubmission; use the preview and submit API")
	}
	obj, err := s.get(ctx, "workflowsubmissions", name)
	if err != nil {
		return fmt.Errorf("cannot read the frozen WorkflowSubmission")
	}
	var receipt parameterapi.WorkflowSubmissionSpec
	if err = decode(obj, &receipt); err != nil {
		return fmt.Errorf("invalid WorkflowSubmission")
	}
	if receipt.Workflow.Name != next.Name || receipt.Workflow.Namespace != next.Namespace || !sameParameterIdentity(next, &receipt.Workflow) || p.Digest(next.Spec) != p.Digest(receipt.Workflow.Spec) {
		return fmt.Errorf("workflow differs from its frozen submission")
	}
	// Reuse the API's schema, binding, action, Secret-reference and source access
	// checks. Forging a receipt does not bypass the definitions' type contracts.
	preview, err := s.Preview(ctx, receipt.Binding, receipt.Revision)
	if err != nil {
		return fmt.Errorf("submission definitions are unavailable or unauthorized")
	}
	if len(preview.Errors) != 0 {
		return fmt.Errorf("submission parameter validation failed")
	}
	if receipt.Definitions != nil && p.Digest(receipt.Definitions) != p.Digest(preview.Definitions) {
		return fmt.Errorf("submission definition snapshot differs from the resolved contract")
	}
	if receipt.ExecutionDigest != preview.ExecutionDigest || next.Annotations[p.Group+"/revision"] != receipt.Revision || next.Annotations[p.Group+"/execution-digest"] != preview.ExecutionDigest || p.Digest(next.Spec) != p.Digest(preview.Workflow.Spec) {
		return fmt.Errorf("submission is stale or differs from the resolved parameter contract")
	}
	return nil
}

func (s Store) validateParameterRetry(ctx context.Context, next, old *wfv1.Workflow) error {
	if !AdoptsParameterization(old) || next.Annotations[p.Group+"/submission"] != old.Annotations[p.Group+"/submission"] || next.Annotations[p.Group+"/execution-digest"] != old.Annotations[p.Group+"/execution-digest"] {
		return fmt.Errorf("cannot replace or remove the frozen submission identity")
	}
	name := next.Annotations[p.Group+"/retry"]
	if name == "" || name == old.Annotations[p.Group+"/retry"] {
		var before, after any
		a, _ := json.Marshal(contractSpec(old))
		b, _ := json.Marshal(contractSpec(next))
		_ = json.Unmarshal(a, &before)
		_ = json.Unmarshal(b, &after)
		paths := []string{}
		for _, change := range p.Diff(before, after, []p.IdentityRule{{Path: "/templates", KeyPath: "/name"}, {Path: "/volumes", KeyPath: "/name"}}) {
			paths = append(paths, change.Path)
			if len(paths) == 8 {
				break
			}
		}
		return fmt.Errorf("execution input changes require a checked ParameterRetry (changed paths: %v)", paths)
	}
	obj, err := s.get(ctx, "parameterretries", name)
	if err != nil {
		return fmt.Errorf("cannot read the ParameterRetry")
	}
	var receipt parameterapi.ParameterRetrySpec
	if err = decode(obj, &receipt); err != nil {
		return fmt.Errorf("invalid ParameterRetry")
	}
	if receipt.Plan == nil || receipt.Workflow != old.Name || receipt.WorkflowUID != string(old.UID) || receipt.ResourceVersion != old.ResourceVersion || receipt.Revision != next.Annotations[p.Group+"/revision"] || receipt.PreviousRevision != old.Annotations[p.Group+"/revision"] {
		return fmt.Errorf("ParameterRetry does not match this workflow and revision")
	}
	var preview *Preview
	if receipt.Revision == receipt.PreviousRevision && len(receipt.Plan.CallPaths) > 0 {
		if err = s.validateWorkflowSecrets(ctx, old); err != nil {
			return err
		}
		definitions, e := s.executionDefinitions(ctx, old)
		if e != nil {
			return e
		}
		preview = &Preview{Workflow: old.DeepCopy(), Definitions: definitions, ExecutionDigest: old.Annotations[p.Group+"/execution-digest"]}
	} else {
		preview, err = s.Preview(ctx, receipt.Plan.Binding, receipt.Revision)
	}
	if err != nil || len(preview.Errors) != 0 {
		return fmt.Errorf("retry parameter contract is unavailable or invalid")
	}
	if receipt.Definitions != nil && p.Digest(receipt.Definitions) != p.Digest(preview.Definitions) {
		return fmt.Errorf("retry definition snapshot differs from the resolved contract")
	}
	if p.Digest(next.Spec) != p.Digest(preview.Workflow.Spec) {
		return fmt.Errorf("retry differs from the resolved parameter contract")
	}
	plan, err := planPartialRetry(ctx, old, preview.Workflow, receipt.Plan.CallPaths...)
	if err != nil {
		return fmt.Errorf("retry does not preserve the supported execution contract")
	}
	if err = s.verifyRetainedArtifacts(ctx, old, plan); err != nil {
		return err
	}
	plan.Binding = receipt.Plan.Binding
	if receipt.Plan.ExecutionDigest != "" {
		plan.ExecutionDigest = preview.ExecutionDigest
	}
	plan.Revision = receipt.Revision
	plan.PreviousRevision = receipt.PreviousRevision
	plan.Digest = p.Digest(plan)
	if p.Digest(plan) != p.Digest(receipt.Plan) {
		return fmt.Errorf("retry impact plan changed")
	}
	for id, fingerprint := range plan.RetainedNodes {
		node, ok := next.Status.Nodes[id]
		if !ok || p.ResultFingerprint(node) != fingerprint {
			return fmt.Errorf("retry invalidates a retained result")
		}
	}
	return nil
}

// Native retry resets a terminal Workflow to a nonterminal phase. Ordinary
// controller status writes, input completion and suspend/resume do not qualify.
func resetsNativeExecution(next, old *wfv1.Workflow) bool {
	terminal := func(phase wfv1.WorkflowPhase) bool {
		return phase == wfv1.WorkflowSucceeded || phase == wfv1.WorkflowFailed || phase == wfv1.WorkflowError
	}
	return old != nil && next != nil && terminal(old.Status.Phase) && !terminal(next.Status.Phase)
}

func (s Store) validateWorkflowSecrets(ctx context.Context, wf *wfv1.Workflow) error {
	definitions, err := s.executionDefinitions(ctx, wf)
	if err != nil {
		return err
	}
	if definitions != nil {
		if err = validatePinnedSecrets(ctx, s.Namespace, definitions.Versions); err != nil {
			return err
		}
	}
	return s.validateInputSecrets(ctx, wf)
}

func (s Store) validateInputSecrets(ctx context.Context, wf *wfv1.Workflow) error {
	for key, name := range wf.Annotations {
		if !strings.HasPrefix(key, p.Group+"/input-") {
			continue
		}
		obj, err := s.get(ctx, "parameternodeinputs", name)
		if err != nil {
			return fmt.Errorf("input receipt is unavailable or unauthorized")
		}
		var receipt parameterapi.ParameterNodeInputSpec
		if err = decode(obj, &receipt); err != nil {
			return err
		}
		if receipt.Workflow != wf.Name || receipt.WorkflowUID != string(wf.UID) || key != p.Group+"/input-"+p.Digest(receipt.NodeID)[:16] {
			return fmt.Errorf("input receipt identity mismatch")
		}
		if err = validatePinnedSecrets(ctx, s.Namespace, receipt.SecretVersions); err != nil {
			return err
		}
	}
	return nil
}
