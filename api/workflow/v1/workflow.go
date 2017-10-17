package v1

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

func (wf *Workflow) Completed() bool {
	return wf.Status == WorkflowStatusSuccess ||
		wf.Status == WorkflowStatusFailed ||
		wf.Status == WorkflowStatusCanceled
}

func (wf *Workflow) DeepCopyObject() runtime.Object {
	wfBytes, err := json.Marshal(wf)
	if err != nil {
		panic(err)
	}
	var copy Workflow
	err = json.Unmarshal(wfBytes, &copy)
	if err != nil {
		panic(err)
	}
	return &copy
}

func (wfl *WorkflowList) DeepCopyObject() runtime.Object {
	wflBytes, err := json.Marshal(wfl)
	if err != nil {
		panic(err)
	}
	var copy WorkflowList
	err = json.Unmarshal(wflBytes, &copy)
	if err != nil {
		panic(err)
	}
	return &copy
}

func (wf *Workflow) GetTemplate(name string) *Template {
	for _, t := range wf.Templates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}
