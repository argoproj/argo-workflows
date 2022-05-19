package v1alpha1

import apiv1 "k8s.io/api/core/v1"

type Job struct {
	Image      string    `json:"image" protobuf:"bytes,1,opt,name=image"`
	WorkingDir string    `json:"workingDir,omitempty" protobuf:"bytes,2,opt,name=workingDir"`
	Steps      []JobStep `json:"steps" protobuf:"bytes,3,rep,name=steps"`
}

func (j Job) GetContainers() []apiv1.Container {
	return []apiv1.Container{{
		Name:       "main",
		Image:      j.Image,
		WorkingDir: j.WorkingDir,
		Command:    []string{"/var/run/argo/argoexec", "job"},
	}}
}

func (in *Job) StepIndex(stepName string) int {
	for i, s := range in.Steps {
		if s.Name == stepName {
			return i
		}
	}
	return -1
}

type JobStep struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Run  string `json:"run" protobuf:"bytes,2,opt,name=run"`
	If   string `json:"if,omitempty" protobuf:"bytes,3,opt,name=if"`
}

func (j JobStep) GetIf() string {
	if j.If != "" {
		return j.If
	}
	return "success()"
}
