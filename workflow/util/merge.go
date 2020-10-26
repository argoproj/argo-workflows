package util

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// MergeTo will merge one workflow (the "patch" workflow) into another (the "target" workflow.
// If the target workflow defines a field, this take precedence over the patch.
func MergeTo(patch, target *wfv1.Workflow) error {
	if target == nil || patch == nil {
		return nil
	}

	patchWfBytes, err := json.Marshal(patch)
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
	err = json.Unmarshal(mergedWfByte, target)
	if err != nil {
		return err
	}
	return nil
}

// MergeMap will merge all element from right map to left map if it is not present in left.
func MergeMap(left, right map[string]string) {
	for key, val := range right {
		if _, ok := left[key]; !ok {
			fmt.Println(key, val)
			left[key] = val
		}
	}
}

// MergeWfSpece will merge the workflow space with order of precedence.
// 1. Workflow Spec, 2 WorkflowTemplate Spec (WorkflowTemplateRef), 3. WorkflowDefault Spec.
func MergeWfSpecs(wfSpec, wftSpec, wfDefaultSpec *wfv1.WorkflowSpec) (*wfv1.Workflow, error) {
	if wfSpec == nil {
		return nil, fmt.Errorf("invalid Workflow spec")
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
	return &targetWf, nil
}

// MergeMetaData will merge the patch metadata into target metadata.
// This func will merge only labels and annotations.
func MergeMetaDataTo(patch, targetMetaData *metav1.ObjectMeta) {
	if patch != nil && patch.Labels != nil {
		if targetMetaData.Labels == nil {
			targetMetaData.Labels = make(map[string]string)
		}
		MergeMap(targetMetaData.Labels, patch.Labels)
	}
	if patch != nil && patch.Annotations != nil {
		if targetMetaData.Annotations == nil {
			targetMetaData.Annotations = make(map[string]string)
		}
		MergeMap(targetMetaData.Annotations, patch.Annotations)
	}
}
