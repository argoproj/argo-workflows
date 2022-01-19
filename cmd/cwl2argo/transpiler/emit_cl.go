package transpiler

import (
	_ "embed"
	"errors"
	"fmt"
	"sort"

	v1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
)

const (
	ArgoType    = "Workflow"
	ArgoVersion = "argoproj.io/v1alpha1"
)

// de-sum typed "CommandlineInputParameter"
type flatCommandlineInputParameter struct {
	Type           Type
	Label          *string
	Value          *string
	SecondaryFiles *SecondaryFiles
	Streamable     *bool
	Doc            Strings
	Id             *string
	Format         *CWLFormat
	LoadContents   *bool
	LoadListing    *LoadListingEnum
	InputBinding   *CommandlineBinding
}

type ParamTranslater interface {
	TranslateToParam(*CommandlineBinding) ([]v1.Parameter, error)
}

func emitDockerRequirement(container *apiv1.Container, d *DockerRequirement) error {
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

func convertAndAdd(inputs *v1.Inputs, input CommandlineInputParameter, addIndexString bool) error {
	// for now we don't care about the other fields
	for i, ty := range input.Type {
		switch ty.Kind {

		case CWLRecordKind:
			fallthrough
		case CWLArrayKind:
			fallthrough
		case CWLEnumKind:
			return errors.New("record|array|enum not yet supported")
		default:
			//noop
		}
		name := *input.Id
		if addIndexString {
			name = fmt.Sprintf("%s_%d", name, i)
		}
		param := v1.Parameter{Name: name}
		inputs.Parameters = append(inputs.Parameters, param)
	}
	return nil
}

func emitInputParam(input CommandlineInputParameter) (*v1.Inputs, error) {
	params := make([]v1.Parameter, 0)
	artifacts := make([]v1.Artifact, 0)
	mappedInput := v1.Inputs{Parameters: params, Artifacts: artifacts}

	if len(input.Type) <= 0 {
		return &mappedInput, nil
	}

	if len(input.Type) == 1 {
		err := convertAndAdd(&mappedInput, input, false)
		if err != nil {
			return nil, err
		}
	} else {
		err := convertAndAdd(&mappedInput, input, true)
		if err != nil {
			return nil, err
		}
	}
	return &mappedInput, nil
}

func dockerNotPresent() error { return errors.New("DockerRequirement was not found") }

func findDockerRequirement(clTool *CommandlineTool) (*DockerRequirement, error) {
	var docker *DockerRequirement
	docker = nil
	for _, req := range clTool.Requirements {
		d, ok := req.(DockerRequirement)
		if ok {
			log.Info("Found DockerRequirement")
			docker = &d
		}
	}

	if docker != nil {
		return docker, nil
	} else {
		return nil, dockerNotPresent()
	}
}

func emitInputParams(template *v1.Template, inputs []CommandlineInputParameter) error {
	params := make([]v1.Parameter, 0)
	artifacts := make([]v1.Artifact, 0)

	for _, input := range inputs {
		newInput, err := emitInputParam(input)
		if err != nil {
			return err
		}
		params = append(params, newInput.Parameters...)
		artifacts = append(artifacts, newInput.Artifacts...)
	}
	mappedInput := v1.Inputs{Parameters: params, Artifacts: artifacts}

	template.Inputs = mappedInput
	return nil
}

// dummy function to evaluate CommandlineTool
// until proper eval functionality is added
func evalArgument(arg CommandlineArgument) (*string, error) {
	switch arg.Kind {
	case ArgumentStringKind:
		return (*string)(&arg.String), nil
	default:
		return nil, errors.New("only string is accepted at the moment")
	}
}

func (inputParameter CommandlineInputParameter) GetInputBindings(inputs map[string]interface{}) ([]flatCommandlineInputParameter, error) {
	bindings := make([]flatCommandlineInputParameter, 0)
	foundTy := false

	if inputParameter.Id == nil {
		return nil, errors.New("input parameter is nil")
	}

	inputi, ok := inputs[*inputParameter.Id]
	if !ok {
		return nil, fmt.Errorf("%s was not present in input", *inputParameter.Id)
	}

	binding := flatCommandlineInputParameter{
		SecondaryFiles: &inputParameter.SecondaryFiles,
		Streamable:     inputParameter.Streamable,
		Doc:            inputParameter.Doc,
		Id:             inputParameter.Id,
		Format:         inputParameter.Format,
		InputBinding:   inputParameter.InputBinding,
	}
	for _, ty := range inputParameter.Type {
		switch ty.Kind {
		case CWLStringKind:
			value, ok := inputi.(string)
			if !ok {
				continue
			}
			binding.Type = CWLStringKind
			binding.Value = &value
			foundTy = true
			bindings = append(bindings, binding)
		default:
			return nil, fmt.Errorf("invalid type %T", inputi)
		}
	}
	if !foundTy {
		return nil, fmt.Errorf("valid type was not present in input for %s", *inputParameter.Id)
	}
	return bindings, nil
}

func sortBindingsByPosition(bindings []flatCommandlineInputParameter) {
	sort.Slice(bindings[:], func(i, j int) bool {
		leftPost := 0
		rightPost := 0
		if bindings[i].InputBinding.Position != nil {
			leftPost = *bindings[i].InputBinding.Position
		}
		if bindings[i].InputBinding.Position != nil {
			rightPost = *bindings[j].InputBinding.Position
		}
		return leftPost < rightPost
	})
}

func emitArgumentParams(container *apiv1.Container,
	inputBindings Inputs,
	baseCommand Strings,
	arguments Arguments,
	inputs map[string]interface{}) ([]flatCommandlineInputParameter, error,
) {
	cmds := make([]string, 0)
	skip := false

	if len(baseCommand) == 0 {
		if len(arguments) == 0 {
			return nil, errors.New("len(baseCommand)==0 && len(arguments)==0")
		}
		cmd, err := evalArgument(arguments[0])
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, *cmd)
		skip = false
	}

	for _, cmd := range baseCommand {
		cmds = append(cmds, cmd)
	}

	for i, arg := range arguments {
		if i == 0 && skip {
			continue
		}
		cmd, err := evalArgument(arg)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, *cmd)
	}

	bindings := make([]flatCommandlineInputParameter, 0)

	for _, inputBinding := range inputBindings {
		newBindings, err := inputBinding.GetInputBindings(inputs)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, newBindings...)
	}
	sortBindingsByPosition(bindings)

	args := make([]string, 0)
	for _, binding := range bindings {
		prefix := ""
		if binding.InputBinding != nil && binding.InputBinding.Prefix != nil {
			sep := true
			if binding.InputBinding.Separate != nil {
				sep = *binding.InputBinding.Separate
			}

			if sep {
				sepArg := *binding.InputBinding.Prefix
				args = append(args, sepArg)
			} else {
				prefix = *binding.InputBinding.Prefix
			}
		}
		arg := fmt.Sprintf("%s{{inputs.parameters.%s}}", prefix, *binding.Id)
		args = append(args, arg)
	}

	container.Command = cmds
	container.Args = args

	return bindings, nil
}

func getInputBindingInputs(inputs Inputs) Inputs {
	newInputs := make(Inputs, 0)

	for _, input := range inputs {
		if input.InputBinding != nil {
			newInputs = append(newInputs, input)
		}
	}
	return newInputs
}

func emitArguments(spec *v1.WorkflowSpec, bindings []flatCommandlineInputParameter) error {
	params := make([]v1.Parameter, 0)
	arts := make([]v1.Artifact, 0)

	for _, binding := range bindings {
		switch binding.Type {
		case CWLStringKind:
			params = append(params, v1.Parameter{Name: *binding.Id, Value: (*v1.AnyString)(binding.Value)})
		default:
			return fmt.Errorf("%T is not supported", binding.Type)
		}
	}
	args := v1.Arguments{Parameters: params, Artifacts: arts}
	spec.Arguments = args
	return nil
}

func EmitCommandlineTool(clTool *CommandlineTool, inputs map[string]interface{}) (*v1.Workflow, error) {
	var wf v1.Workflow
	var err error

	wf.Name = *clTool.Id
	spec := v1.WorkflowSpec{}
	wf.APIVersion = ArgoVersion
	wf.Kind = ArgoType

	container := apiv1.Container{}

	dockerRequirement, err := findDockerRequirement(clTool)
	if err != nil {
		return nil, err
	}

	err = emitDockerRequirement(&container, dockerRequirement)
	if err != nil {
		return nil, err
	}

	template := v1.Template{}
	template.Container = &container
	template.Name = *clTool.Id

	err = emitInputParams(&template, clTool.Inputs)
	if err != nil {
		return nil, err
	}

	inputBindings := getInputBindingInputs(clTool.Inputs)
	bindings, err := emitArgumentParams(&container, inputBindings, clTool.BaseCommand, clTool.Arguments, inputs)
	if err != nil {
		return nil, err
	}

	emitArguments(&spec, bindings)

	spec.Templates = []v1.Template{template}
	spec.Entrypoint = template.Name

	wf.Spec = spec
	return &wf, nil
}
