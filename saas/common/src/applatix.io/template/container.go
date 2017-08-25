package template

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"applatix.io/axerror"
)

// Valid kubernetes pull policies
const (
	PullPolicyIfNotPresent = "IfNotPresent"
	PullPolicyAlways       = "Always"
	PullPolicyNever        = "Never"
)

// NOTE: squash is a mapstructure struct tag but we teach mapstructure to parse the json tags
type ContainerTemplate struct {
	BaseTemplate    `json:",squash"`
	Inputs          *Inputs             `json:"inputs,omitempty"`
	Outputs         *Outputs            `json:"outputs,omitempty"`
	Resources       *ContainerResources `json:"resources,omitempty"`
	Image           string              `json:"image,omitempty"`
	Command         []string            `json:"command,omitempty"`
	Args            []string            `json:"args,omitempty"`
	Env             []NameValuePair     `json:"env,omitempty"`
	LivenessProbe   *ContainerProbe     `json:"liveness_probe,omitempty"`
	ReadinessProbe  *ContainerProbe     `json:"readiness_probe,omitempty"`
	ImagePullPolicy *string             `json:"image_pull_policy,omitempty"`
	Annotations     map[string]string   `json:"annotations,omitempty"`
}

type ContainerResources struct {
	MemMiB   NumberOrString `json:"mem_mib,omitempty"`
	CPUCores NumberOrString `json:"cpu_cores,omitempty"`
}

// ContainerProbe, this could be used for both liveness and readiness probes
type ContainerProbe struct {
	InitialDelaySeconds int                        `json:"initial_delay_seconds,omitempty"`
	TimeoutSeconds      int                        `json:"timeout_seconds,omitempty"`
	PeriodSeconds       int                        `json:"period_seconds,omitempty"`
	FailureThreshold    int                        `json:"failure_threshold,omitempty"`
	SuccessThreshold    int                        `json:"success_threshold,omitempty"`
	Exec                *ContainerProbeExec        `json:"exec,omitempty"`
	HTTPGet             *ContainerProbeHttpRequest `json:"http_get,omitempty"`
}

type ContainerProbeExec struct {
	Command string `json:"command,omitempty"`
}

type ContainerProbeHttpRequest struct {
	Path        string           `json:"path,omitempty"`
	Port        int              `json:"port,omitempty"`
	HTTPHeaders []*NameValuePair `json:"http_headers,omitempty"`
}

func (tmpl *ContainerTemplate) GetInputs() *Inputs {
	return tmpl.Inputs
}

func (tmpl *ContainerTemplate) GetOutputs() *Outputs {
	return tmpl.Outputs
}

func (tmpl *ContainerTemplate) Validate(preproc ...bool) *axerror.AXError {
	preprocess := len(preproc) > 0 && preproc[0]
	if axErr := tmpl.BaseTemplate.Validate(); axErr != nil {
		return axErr
	}
	if tmpl.Image == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Container must have an image specified")
	}
	// TODO: validate it looks like an image

	if tmpl.Resources != nil {
		axErr := tmpl.Resources.Validate()
		if axErr != nil {
			return axErr
		}
	}

	if tmpl.Env != nil {
		for _, env := range tmpl.Env {
			if env.Name == nil || strings.TrimSpace(*env.Name) == "" {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Empty or missing container env name")
			}
			if env.Value == nil {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Missing container env value for %s", *env.Name)
			}
		}
	}

	if tmpl.ImagePullPolicy != nil {
		if *tmpl.ImagePullPolicy != PullPolicyAlways && *tmpl.ImagePullPolicy != PullPolicyIfNotPresent && *tmpl.ImagePullPolicy != PullPolicyNever {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Invalid 'image_pull_policy': %s. Valid options: Always, IfNotPresent, Never", *tmpl.ImagePullPolicy)
		}
	}

	if tmpl.Inputs != nil {
		axErr := tmpl.Inputs.Validate(true)
		if axErr != nil {
			return axErr
		}
		if tmpl.Inlined() {
			for artRef, art := range tmpl.Inputs.Artifacts {
				if art.From == "" {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("inputs.artifacts.%s.from is required for inlined containers", artRef)
				}
				if !IsParam(art.From) {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("inputs.artifacts.%s.from: '%s' has invalid variable format (e.g. %%%%steps.STEP_NAME.outputs.artifacts.ART_NAME%%%%)", artRef, art.From)
				}
			}
			for volRef, vol := range tmpl.Inputs.Volumes {
				if vol.From == "" {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("inputs.volumes.%s.from is required for inlined containers", volRef)
				}
				if !IsParam(vol.From) {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("inputs.volumes.%s.from: '%s' has invalid variable variable format (e.g. %%%%volumes.VOL_NAME%%%%)", volRef, vol.From)
				}
			}
			// inlined containers do not have a reason to have inputs.parameters/fixtures
			// since they are accessible directly from current scope. Allowing them causes
			// problems when we want to reverse inline the container.
			if len(tmpl.Inputs.Fixtures) > 0 || len(tmpl.Inputs.Parameters) > 0 {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessage("inlined containers can only have 'artifacts' and 'volumes' as inputs")
			}
		}
	}

	if tmpl.Outputs != nil && tmpl.Outputs.Artifacts != nil {
		for refName, art := range tmpl.Outputs.Artifacts {
			if art.Path == "" {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("outputs.artifacts.%s 'path' field is required", refName)
			}
			if art.From != "" {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("outputs.artifacts.%s 'from' field is only valid in workflow templates, not container", refName)
			}
		}
	}

	if !preprocess && !tmpl.Inlined() {
		// skip validation of parameter scope during preprocessing because after substitution,
		// there can be unresolved variables. also skip for inlined containers because they will
		// reference variables in the outer workflow/deployment
		axErr := tmpl.ValidateParameterScope()
		if axErr != nil {
			return axErr
		}
	}

	return nil
}

// Inlined returns whether or not this container is inlined or not.
// This is determined if we have a template name or not. We can trust name will be blank since the
// check in InlineContainerTemplateRef.UnmarshalJSON() will ensure this.
func (tmpl *ContainerTemplate) Inlined() bool {
	return strings.TrimSpace(tmpl.Name) == ""
}

// ValidateParameterScope checks that all used parameters within the scope of the template, are declared and of the same type
func (tmpl *ContainerTemplate) ValidateParameterScope() *axerror.AXError {
	declaredParams := getParameterDeclarations(tmpl.Inputs, nil, nil)
	usedParams, axErr := tmpl.usedParameters()
	if axErr != nil {
		return axErr
	}
	axErr = validateParams(declaredParams, usedParams)
	if axErr != nil {
		return axErr
	}
	return nil
}

// ValidateContext validates the context of a container
// Containers are the lowest level building block and do not reference any other templates.
// Thus they can be validated statically without any context. So this is a noop.
func (tmpl *ContainerTemplate) ValidateContext(context *TemplateBuildContext) *axerror.AXError {
	return nil
}

func (r *ContainerResources) Validate() *axerror.AXError {
	if r != nil {
		if !IsParam(string(r.CPUCores)) {
			cpuCores, axErr := r.CPUCoresValue()
			if axErr != nil {
				return axErr
			}
			if cpuCores-0.0 <= 0.000001 {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("invalid cpu_cores: %v. Must be between 0 and 100", r.CPUCores)
			}
		}
		if !IsParam(string(r.MemMiB)) {
			memMiB, axErr := r.MemMiBValue()
			if axErr != nil {
				return axErr
			}
			if memMiB-0.0 <= 0.000001 {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("invalid mem_mib: %v. Must be greater than 0", r.MemMiB)
			}
		}
	}
	return nil
}

// usedParameters detects all the parameters used in various parts of a container template and returns it in a paramMap.
// For containers, this is command, args, env, image, annotations.
// If we are inlined, we also need to include any used parameters in the inputs such as default values.
func (tmpl *ContainerTemplate) usedParameters() (paramMap, *axerror.AXError) {
	pMap := make(paramMap)
	axErr := pMap.extractUsedParams(tmpl.Command, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.Args, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.Env, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.Image, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.Resources, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.Annotations, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	if tmpl.Inlined() && tmpl.Inputs != nil {
		// If we are inlined, also check used parameters in the inputs
		inPmap, axErr := tmpl.Inputs.usedParameters()
		if axErr != nil {
			return nil, axErr
		}
		axErr = pMap.merge(inPmap)
		if axErr != nil {
			return nil, axErr
		}
	}
	return pMap, nil
}

// Copy returns a copy of this template
func (tmpl *ContainerTemplate) Copy() *ContainerTemplate {
	bytes, err := json.Marshal(tmpl)
	if err != nil {
		panic(err)
	}
	var copy ContainerTemplate
	err = json.Unmarshal(bytes, &copy)
	if err != nil {
		panic(err)
	}
	return &copy
}

// Substitute returns a copy of this template with all instances of `from`` with `to`
func (tmpl *ContainerTemplate) Substitute(from string, to string) *ContainerTemplate {
	bytes, err := json.Marshal(tmpl)
	if err != nil {
		panic(err)
	}
	newStr := strings.Replace(string(bytes), from, to, -1)
	var copy ContainerTemplate
	err = json.Unmarshal([]byte(newStr), &copy)
	if err != nil {
		panic(err)
	}
	return &copy
}

// ReverseInline converts an inlined container to a regular container template. It ensures that
// any referenced parameters from parent's scope are passed as arguments.
func (tmpl *ContainerTemplate) ReverseInline(name string) (*ContainerTemplate, Arguments, *axerror.AXError) {
	if !tmpl.Inlined() {
		return nil, nil, axerror.ERR_AX_INTERNAL.NewWithMessage("ReverseInline() called against a container which is not inlined")
	}
	reversed := tmpl.Copy()
	reversed.Name = name
	reversed.Type = TemplateTypeContainer
	reversed.Version = TemplateVersion1
	arguments := make(Arguments)

	// First, null out the .From fields for inlined volumes & artifacts, since 'from'
	// is an inlined container concept and we are now explictly passing them as arguments
	// We want to do this before we extract the parameter usages.
	if reversed.Inputs != nil {
		for artRef, art := range reversed.Inputs.Artifacts {
			if art.From != "" && !strings.HasPrefix(art.From, "%%artifacts.") {
				// only pass as argument, null it out if we are referencing steps/inputs, and not from global scope %%artifacts.tag.<tagname>.<artname>%%
				newFrom := art.From
				arguments["artifacts."+artRef] = &newFrom
				art.From = ""
			}
		}
		for volRef, vol := range reversed.Inputs.Volumes {
			newFrom := vol.From
			arguments["volumes."+volRef] = &newFrom
			vol.From = ""
		}
	}

	usedParams, axErr := reversed.usedParameters()
	if axErr != nil {
		return nil, nil, axErr
	}
	if len(usedParams) == 0 {
		// no parameter usages detected. nothing more to do
		return reversed, arguments, nil
	}
	for _, p := range usedParams {
		if p.paramType == paramTypeFixture {
			return nil, nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Parameter usages of type %s (%%%%%s%%%%) not yet supported", p.paramType, p.name)
		}
	}
	if reversed.Inputs == nil {
		reversed.Inputs = &Inputs{}
	}

	// The logic below iterates the parameter references in the template. It adds each parameter
	// as an input parameter, as well as builds up the argument map to be sent to the reverse
	// inlined container. It handles name collisions that occur when there is a collision in input names.

	// First handle any references to %%inputs...%%. For input references like "%%inputs.type.name%% the job is easy:
	// We simply need to add it to our own inputs section, and set them as arguments. We process these first.
	for _, p := range usedParams {
		if !strings.HasPrefix(p.name, "inputs.") {
			continue
		}
		argName := strings.TrimPrefix(p.name, "inputs.")
		if _, ok := arguments[argName]; ok {
			// we already handled input artifacts/volumes above
			continue
		}
		pName := "%%" + p.name + "%%"
		arguments[argName] = &pName
		parts := strings.Split(p.name, ".")
		inputName := parts[len(parts)-1]
		switch p.paramType {
		case paramTypeString:
			if reversed.Inputs.Parameters == nil {
				reversed.Inputs.Parameters = make(map[string]*InputParameter)
			}
			reversed.Inputs.Parameters[inputName] = nil
		case paramTypeFixture:
			if reversed.Inputs.Fixtures == nil {
				reversed.Inputs.Fixtures = make(map[string]*InputFixture)
			}
			reversed.Inputs.Fixtures[inputName] = nil
		default:
			return nil, nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Parameter usages of type %s (%%%%%s%%%%) not yet supported", p.paramType, p.name)
		}
	}

	// for other param usages like "%%fixtures.myfixture%%" we have to generate a corresponding input name.
	// (e.g. inputs.fixtures.myfixture. Hopefully the name does not collide with an existing input. But if
	// it does, we need to generate a different name. The logic is to append -X until something is found.
	for _, p := range usedParams {
		if strings.HasPrefix(p.name, "inputs.") {
			continue
		}
		inputName := reverseInput(p, reversed.Inputs, arguments)
		// TODO: This substitution is flawed as it could match unexpected strings.
		// correct way is to do a regex match using sub group. Fix later -Jesse
		reversed = reversed.Substitute("%%"+p.name+".", "%%"+inputName+".")
		reversed = reversed.Substitute("%%"+p.name+"%%", "%%"+inputName+"%%")
	}

	// This check is for internal use, to make sure the template we are generating is still valid
	axErr = reversed.Validate()
	if axErr != nil {
		return nil, nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Failed to reverse inline (%s): %v", tmpl.Name, axErr)
	}
	return reversed, arguments, nil
}

// reverseInput finds an available parameter name to use when reverse inlining a container
// if there is a name collision, it appends -1, -2, etc... to the param name until one is found
func reverseInput(p param, inputs *Inputs, arguments Arguments) string {
	if strings.HasPrefix(p.name, "inputs.") {
		panic("setInputName called on an param thats already in inputs " + p.name)
	}
	argName := strings.Replace(p.name, ".", "_", -1)
	counter := 0
	paramName := "%%" + p.name + "%%"
	for {
		var suffix string
		if counter == 0 {
			suffix = ""
		} else {
			suffix = fmt.Sprintf("-%d", counter)
		}
		candidate := argName + suffix
		switch p.paramType {
		case paramTypeString:
			if inputs.Parameters == nil {
				inputs.Parameters = make(map[string]*InputParameter)
			}
			_, exists := inputs.Parameters[candidate]
			if !exists {
				inputs.Parameters[candidate] = nil
				arguments["parameters."+candidate] = &paramName
				return "inputs.parameters." + candidate
			}
		case paramTypeFixture:
			if inputs.Fixtures == nil {
				inputs.Fixtures = make(map[string]*InputFixture)
			}
			_, exists := inputs.Fixtures[candidate]
			if !exists {
				inputs.Fixtures[candidate] = nil
				arguments["fixtures."+candidate] = &paramName
				return "inputs.fixtures." + candidate
			}
		case paramTypeVolume:
			if inputs.Volumes == nil {
				inputs.Volumes = make(map[string]*InputVolume)
			}
			_, exists := inputs.Volumes[candidate]
			if !exists {
				inputs.Volumes[candidate] = nil
				arguments["volumes."+candidate] = &paramName
				return "inputs.volumes." + candidate
			}
		default:
			panic(p)
		}
		counter++
	}
}

func (cr *ContainerResources) MemMiBValue() (float64, *axerror.AXError) {
	val, err := strconv.ParseFloat(string(cr.MemMiB), 64)
	if err != nil {
		return 0, axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("'%s' is not a valid number", cr.MemMiB)
	}
	return val, nil
}

func (cr *ContainerResources) CPUCoresValue() (float64, *axerror.AXError) {
	val, err := strconv.ParseFloat(string(cr.CPUCores), 64)
	if err != nil {
		return 0, axerror.ERR_AX_ILLEGAL_ARGUMENT.NewWithMessagef("'%s' is not a valid number", cr.CPUCores)
	}
	return val, nil
}
