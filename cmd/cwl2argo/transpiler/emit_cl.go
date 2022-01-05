package transpiler

import (
	_ "embed"
	"errors"

	v1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
)

func emitDockerRequirement(d *DockerRequirement, container *apiv1.Container) error {
	tmpContainer := container.DeepCopy()

	if d.DockerPull == nil {
		return errors.New("dockerPull is a required field")
	}

	tmpContainer.Image = *d.DockerPull

	if d.DockerFile != nil {
		return errors.New("")
	}

	if d.DockerImageId != nil {
		return errors.New("")
	}

	if d.DockerImport != nil {
		return errors.New("")
	}

	*container = *tmpContainer
	return nil
}

func emitArgument(input CommandlineInputParameter) []v1.Parameter {
	params := make([]v1.Parameter, 0)
	ctx := NewFlattenContext()

	for _, ty := range input.Type {
		ty.flatten(&ctx, *input.Id)
	}
	flatTypes := ctx.GetFlatTypes()
	for name, ty := range flatTypes {
		switch ty.(type) {
		case CommandlineInputEnumSchema:
			bottom()
		default:
			param := v1.Parameter{}
			param.Name = name
			params = append(params, param)
		}
	}
	return params
}

func dockerNotPresent() error { return errors.New("DockerRequirement was not found") }

func findDockerRequirement(clTool *CommandlineTool) (*DockerRequirement, error) {
	var docker *DockerRequirement
	docker = nil
	for _, req := range clTool.Requirements {
		d, ok := req.(*DockerRequirement)
		if ok {
			docker = d
		}
	}
	if docker != nil {
		return docker, nil
	} else {
		return nil, dockerNotPresent()
	}
}

func emitArguments(inputs []CommandlineInputParameter) []v1.Parameter {
	params := make([]v1.Parameter, 0)
	for _, input := range inputs {
		params = append(emitArgument(input), params...)
	}

	return params
}

func EmitArgo(clTool *CommandlineTool) (*v1.Workflow, error) {
	var wf v1.Workflow
	var err error

	spec := v1.WorkflowSpec{}
	container := apiv1.Container{}

	dockerRequirement, err := findDockerRequirement(clTool)
	if err != nil {
		return nil, err
	}

	err = emitDockerRequirement(dockerRequirement, &container)
	if err != nil {
		return nil, err
	}

	template := v1.Template{}
	template.Container = &container
	// params := emitArguments(clTool.Inputs)

	spec.Templates = []v1.Template{template}

	wf.Spec = spec
	return &wf, nil
}
