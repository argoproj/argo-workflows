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

func MergeMap(left, right map[string]string) {
	for key, val := range right {
		if _, ok := left[key]; !ok {
			fmt.Println(key, val)
			left[key] = val
		}
	}
}

func MergeMetaDatato(patch, targetMetaData *metav1.ObjectMeta) {
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
