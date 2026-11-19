package common

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/util/strategicpatch"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// MergeWorkflowSpecs merges specs, highest priority first, as a workflow is
// merged with its WorkflowTemplate and the workflow defaults: each spec only
// fills in what the ones before it leave unset (MergeTo). Nil specs are
// skipped, and it returns nil if the first is nil.
func MergeWorkflowSpecs(specs ...*wfv1.WorkflowSpec) (*wfv1.WorkflowSpec, error) {
	if len(specs) == 0 || specs[0] == nil {
		return nil, nil
	}
	merged := &wfv1.Workflow{Spec: *specs[0].DeepCopy()}
	for _, spec := range specs[1:] {
		if spec == nil {
			continue
		}
		if err := MergeTo(&wfv1.Workflow{Spec: *spec.DeepCopy()}, merged); err != nil {
			return nil, err
		}
	}
	return &merged.Spec, nil
}

// MergeTo will merge one workflow (the "patch" workflow) into another (the "target" workflow.
// If the target workflow defines a field, this take precedence over the patch.
func MergeTo(patch, target *wfv1.Workflow) error {
	if target == nil || patch == nil {
		return nil
	}

	// Temporarily remove hooks and labelsFrom as they don't merge
	patchHooks := patch.Spec.Hooks
	patch.Spec.Hooks = nil
	var patchLabelsFrom map[string]wfv1.LabelValueFrom
	if patch.Spec.WorkflowMetadata != nil {
		patchLabelsFrom = patch.Spec.WorkflowMetadata.LabelsFrom
		patch.Spec.WorkflowMetadata.LabelsFrom = nil
	}

	patchWfBytes, err := json.Marshal(patch)
	patch.Spec.Hooks = patchHooks
	if len(patchLabelsFrom) != 0 {
		patch.Spec.WorkflowMetadata.LabelsFrom = patchLabelsFrom
	}
	if err != nil {
		return err
	}

	targetWfByte, err := json.Marshal(target)
	if err != nil {
		return err
	}
	var mergedWfByte []byte

	mergedWfByte, err = strategicpatch.StrategicMergePatch(patchWfBytes, targetWfByte, wfv1.Workflow{})

	if err != nil {
		return err
	}

	target.Spec = wfv1.WorkflowSpec{}
	err = json.Unmarshal(mergedWfByte, target)
	if err != nil {
		return err
	}

	if len(patchHooks) != 0 && target.Spec.Hooks == nil {
		target.Spec.Hooks = make(wfv1.LifecycleHooks)
	}
	for name, hook := range patchHooks {
		// If the patch hook doesn't exist in target
		if _, ok := target.Spec.Hooks[name]; !ok {
			target.Spec.Hooks[name] = hook
		}
	}

	if len(patchLabelsFrom) != 0 && target.Spec.WorkflowMetadata.LabelsFrom == nil {
		target.Spec.WorkflowMetadata.LabelsFrom = make(map[string]wfv1.LabelValueFrom)
	}
	for key, val := range patchLabelsFrom {
		// If the patch labelFrom doesn't exist in target
		if _, ok := target.Spec.WorkflowMetadata.LabelsFrom[key]; !ok {
			target.Spec.WorkflowMetadata.LabelsFrom[key] = val
		}
	}
	return nil
}
