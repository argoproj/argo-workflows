package common

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/util/strategicpatch"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// MergeTo will merge one workflow (the "patch" workflow) into another (the "target" workflow.
// If the target workflow defines a field, this take precedence over the patch.
func MergeTo(patch, target *wfv1.Workflow) error {
	if target == nil || patch == nil {
		return nil
	}

	// Hooks and labelsFrom don't merge, so they are left out of the patch that is marshalled and added to the result
	// afterwards. The patch can be shared between goroutines, e.g. the workflow defaults of the controller, so work on
	// a shallow copy and never write to the patch itself.
	patchHooks := patch.Spec.Hooks
	var patchLabelsFrom map[string]wfv1.LabelValueFrom
	patchWithoutUnmergeable := *patch
	patchWithoutUnmergeable.Spec.Hooks = nil
	if patch.Spec.WorkflowMetadata != nil {
		patchLabelsFrom = patch.Spec.WorkflowMetadata.LabelsFrom
		workflowMetadata := *patch.Spec.WorkflowMetadata
		workflowMetadata.LabelsFrom = nil
		patchWithoutUnmergeable.Spec.WorkflowMetadata = &workflowMetadata
	}

	patchWfBytes, err := json.Marshal(&patchWithoutUnmergeable)
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
