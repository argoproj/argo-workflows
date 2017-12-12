package v1

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/runtime"
)

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
	for _, t := range wf.Spec.Templates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// NodeID creates a deterministic node ID based on a node name
func (wf *Workflow) NodeID(name string) string {
	if name == wf.ObjectMeta.Name {
		return wf.ObjectMeta.Name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return fmt.Sprintf("%s-%v", wf.ObjectMeta.Name, h.Sum32())
}

func (t *Template) DeepCopy() *Template {
	tBytes, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	var copy Template
	err = json.Unmarshal(tBytes, &copy)
	if err != nil {
		panic(err)
	}
	return &copy
}

func (s *WorkflowStep) DeepCopy() *WorkflowStep {
	bytes, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	var copy WorkflowStep
	err = json.Unmarshal(bytes, &copy)
	if err != nil {
		panic(err)
	}
	return &copy
}
