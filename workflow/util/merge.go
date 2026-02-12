package util

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

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

// mergeMap will merge all element from right map to left map if it is not present in left.
func mergeMap(from, to map[string]string) {
	for key, val := range from {
		if _, ok := to[key]; !ok {
			to[key] = val
		}
	}
}

// JoinWorkflowMetaData will join the workflow metadata with the following order of preference
// 1. Workflow, 2 WorkflowTemplate (WorkflowTemplateRef), 3. WorkflowDefault.
func JoinWorkflowMetaData(wfMetaData, wfDefaultMetaData *metav1.ObjectMeta) {
	if wfDefaultMetaData != nil {
		mergeMetaDataTo(wfDefaultMetaData, wfMetaData)
	}
}

// JoinWorkflowSpec will join the workflow specs with the following order of preference
// 1. Workflow Spec, 2 WorkflowTemplate Spec (WorkflowTemplateRef), 3. WorkflowDefault Spec.
func JoinWorkflowSpec(wfSpec, wftSpec, wfDefaultSpec *wfv1.WorkflowSpec) (*wfv1.Workflow, error) {
	if wfSpec == nil {
		return nil, nil
	}
	targetWf := wfv1.Workflow{Spec: *wfSpec.DeepCopy()}
	if wftSpec != nil {
		err := MergeTo(&wfv1.Workflow{Spec: *wftSpec.DeepCopy()}, &targetWf)
		if err != nil {
			return nil, err
		}
	}
	if wfDefaultSpec != nil {
		err := MergeTo(&wfv1.Workflow{Spec: *wfDefaultSpec.DeepCopy()}, &targetWf)
		if err != nil {
			return nil, err
		}
	}

	// This condition will update the workflow Spec suspend value if merged value is different.
	// This scenario will happen when Workflow with WorkflowTemplateRef has suspend template
	if wfSpec.Suspend != targetWf.Spec.Suspend {
		targetWf.Spec.Suspend = wfSpec.Suspend
	}
	// The value of an argument should respect the priority order of the merge.
	// However, if Value, ValueFrom and Default are set in different places,
	// i.e. the Workflow, WorflowTemplate and WorkflowDefault, we should only keep the one with the greatest priority.
	wfParamsMap := parametersToMapByName(wfSpec)
	wftParamsMap := parametersToMapByName(wftSpec)
	for index, parameter := range targetWf.Spec.Arguments.Parameters {
		if parameter.HasValue() {
			if param, ok := wfParamsMap[parameter.Name]; ok {
				targetWf.Spec.Arguments.Parameters[index] = *param.DeepCopy()
			} else if param, ok := wftParamsMap[parameter.Name]; ok {
				targetWf.Spec.Arguments.Parameters[index] = *param.DeepCopy()
			}
			// If none of the above conditions are met, we can safely assume that the parameter is set from the wfDefaultSpec.
		}
	}

	// Apply the same priority logic for artifacts.
	// Artifacts from the workflow spec should take precedence over those from the template.
	wfArtifactsMap := artifactsToMapByName(wfSpec)
	wftArtifactsMap := artifactsToMapByName(wftSpec)
	for index, artifact := range targetWf.Spec.Arguments.Artifacts {
		if art, ok := wfArtifactsMap[artifact.Name]; ok {
			targetWf.Spec.Arguments.Artifacts[index] = *art.DeepCopy()
		} else if art, ok := wftArtifactsMap[artifact.Name]; ok {
			targetWf.Spec.Arguments.Artifacts[index] = *art.DeepCopy()
		}
		// If none of the above conditions are met, we can safely assume that the artifact is set from the wfDefaultSpec.
	}

	return &targetWf, nil
}

func parametersToMapByName(spec *wfv1.WorkflowSpec) map[string]wfv1.Parameter {
	parameterMap := make(map[string]wfv1.Parameter)
	if spec != nil {
		for _, param := range spec.Arguments.Parameters {
			if param.HasValue() {
				parameterMap[param.Name] = param
			}
		}
	}
	return parameterMap
}

func artifactsToMapByName(spec *wfv1.WorkflowSpec) map[string]wfv1.Artifact {
	artifactMap := make(map[string]wfv1.Artifact)
	if spec != nil {
		for _, artifact := range spec.Arguments.Artifacts {
			artifactMap[artifact.Name] = artifact
		}
	}
	return artifactMap
}

// mergeMetadata will merge the labels and annotations into the target metadata.
func mergeMetaDataTo(from, to *metav1.ObjectMeta) {
	if from == nil {
		return
	}
	if from.Labels != nil {
		if to.Labels == nil {
			to.Labels = make(map[string]string)
		}
		mergeMap(from.Labels, to.Labels)
	}
	if from.Annotations != nil {
		if to.Annotations == nil {
			to.Annotations = make(map[string]string)
		}
		mergeMap(from.Annotations, to.Annotations)
	}
}
