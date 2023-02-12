package v1alpha1

import apiv1 "k8s.io/api/core/v1"

// Job is a template for a job resource
type Job struct {
	// Image is the container image to run
	Image string `json:"image" protobuf:"bytes,1,opt,name=image"`
	// WorkingDir is the working directory to run the job in
	WorkingDir string `json:"workingDir,omitempty" protobuf:"bytes,2,opt,name=workingDir"`
	// Steps is the list of steps to run
	Steps []JobStep `json:"steps" protobuf:"bytes,3,rep,name=steps"`
}

// GetContainers returns the list of containers to run
func (j Job) GetContainers() []apiv1.Container {
	return []apiv1.Container{{
		Name:       "main",
		Image:      j.Image,
		WorkingDir: j.WorkingDir,
		Command:    []string{"/var/run/argo/argoexec", "job"},
	}}
}

// StepIndex returns the index of the step with the given name
func (in *Job) StepIndex(stepName string) int {
	for i, s := range in.Steps {
		if s.Name == stepName {
			return i
		}
	}
	panic("step not found")
}

// JobStep is a step in a job
type JobStep struct {
	// Name is the name of the step, must be unique within the job
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Run is the shell script to run.
	Run string `json:"run" protobuf:"bytes,2,opt,name=run"`
	// If is the expression to evaluate to determine if the step should run, default "success()"
	If string `json:"if,omitempty" protobuf:"bytes,3,opt,name=if"`
}

// GetIf returns the expression to evaluate to determine if the step should run
func (j JobStep) GetIf() string {
	if j.If != "" {
		return j.If
	}
	return "success()"
}
