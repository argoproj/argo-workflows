package util

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/util/strategicpatch"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// MergeTo will do strategic merge the workflows
// patch workflow will be merged into target workflow.
// Target value will not be overwrite if it is already present
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
