package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"applatix.io/axerror"
	"applatix.io/common"
	"github.com/mitchellh/mapstructure"
)

// All Argo template types
const (
	TemplateTypeContainer  = "container"
	TemplateTypeWorkflow   = "workflow"
	TemplateTypeDeployment = "deployment"
	TemplateTypeFixture    = "fixture"
	TemplateTypePolicy     = "policy"
	TemplateTypeProject    = "project"
)

// Supported template versions
const (
	TemplateVersion1 = "1"
)

// Reserved keywords used in service template parameters
const (
	KeywordSession    = "session"
	KeywordFixtures   = "fixtures"
	KeywordAttributes = "attributes"
	KeywordVolumes    = "volumes"
	KeywordArtifacts  = "artifacts"
	KeywordConfig     = "config"
)

// Types of inputs to template
const (
	InputTypeParameters = "parameters"
	InputTypeArtifacts  = "artifacts"
	InputTypeVolumes    = "volumes"
	InputTypeFixtures   = "fixtures"
)

// NumberOrString is a custom type which allows us to unmarshal a number as a string
type NumberOrString string

func (a *NumberOrString) UnmarshalJSON(b []byte) error {
	var val interface{}
	json.Unmarshal(b, &val)
	switch val.(type) {
	case float64:
		str := fmt.Sprintf("%v", val)
		*a = NumberOrString(str)
	case string:
		str := val.(string)
		*a = NumberOrString(str)
	default:
		return fmt.Errorf("Cannot convert '%s' to string", string(b))
	}
	return nil
}

// BaseTemplate is the base structure for all Argo templates
type BaseTemplate struct {
	Type        string            `json:"type,omitempty"`
	Version     NumberOrString    `json:"version,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	ID          string            `json:"id,omitempty"`
	Repo        string            `json:"repo,omitempty"`
	Branch      string            `json:"branch,omitempty"`
	Revision    string            `json:"revision,omitempty"`
}

type TemplateIf interface {
	GetType() string
	GetVersion() string
	GetName() string
	GetInputs() *Inputs
	GetLabels() map[string]string
	GetOutputs() *Outputs
	// Validate performs template validation against all fields of the template.
	// The preprocess flag indicates if the validate method is being called in
	// the context of revalidating the template during preprocessing. There may
	// be different or stricture validation rules in the context of preprocessing.
	// (e.g. after argument substitution has been performed)
	Validate(preprocess ...bool) *axerror.AXError
	// ValidateContext performs template validation with respect to other templates with respect to compatible tempate types and parameter agreement
	ValidateContext(context *TemplateBuildContext) *axerror.AXError
	String() string
	setRepoInfo(repo, branch, revision string)
}

// TemplateRef is a reference to another service template (e.g. when a workflow step references another template in its step)
type TemplateRef struct {
	Template  string    `json:"template,omitempty"`
	Arguments Arguments `json:"arguments,omitempty"`
}

type Arguments map[string]*string

func (args *Arguments) Copy() *Arguments {
	bytes, err := json.Marshal(args)
	if err != nil {
		panic(fmt.Sprintf("Error copying arguments when marshaling, err %v", err))
	}
	var newArguments Arguments
	err = json.Unmarshal(bytes, &newArguments)
	if err != nil {
		panic(fmt.Sprintf("Error copying arguments when unmarshaling, err %v", err))
	}
	return &newArguments
}

// InlineContainerTemplateRef can act as a template ref to another template, or a fully inlined container yaml
// NOTE: squash is a mapstructure struct tag but we teach mapstructure to parse the json tags
type InlineContainerTemplateRef struct {
	TemplateRef       `json:",squash"`
	ContainerTemplate `json:",squash"`
}

// NameValuePair is a generic data structure for holding a name value pair
type NameValuePair struct {
	Name  *string `json:"name"`
	Value *string `json:"value"`
}

// Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another
type Inputs struct {
	Parameters map[string]*InputParameter `json:"parameters,omitempty"`
	Artifacts  map[string]*InputArtifact  `json:"artifacts,omitempty"`
	Volumes    map[string]*InputVolume    `json:"volumes,omitempty"`
	Fixtures   map[string]*InputFixture   `json:"fixtures,omitempty"`
}

type Input struct {
	Description string `json:"description,omitempty"`
}

// InputParameter indicate a passed string parameter to a service template with an optional default value
type InputParameter struct {
	Input   `json:",squash"`
	Default *string       `json:"default"`
	Options []interface{} `json:"options,omitempty"` // TODO: implement validation
	Regex   string        `json:"regex,omitempty"`   // TODO: implement validation
}

// InputFixture indicate a passed string parameter to a service template
type InputFixture struct {
	Input `json:",squash"`
}

// InputArtifact is an artifact accepted as an input
// In the context of a container template, 'path' indicates where the artifact should be mounted
// In the context of an inlined container, 'from' describes the source of the artifact
type InputArtifact struct {
	Input `json:",squash"`
	Path  string `json:"path,omitempty"`
	From  string `json:"from,omitempty"`
	//Tag  string `json:"tag,omitempty"` // ??????? is this used ??????? -Jesse
}

// InputVolume is a volume accepted as an input
// In the context of a container template, 'mount_path' indicates where the volume should be mounted
// In the context of an inlined container, 'from' describes the source of the artifact
type InputVolume struct {
	Input     `json:",squash"`
	From      string      `json:"from,omitempty" `
	MountPath string      `json:"mount_path,omitempty"`
	Details   interface{} `json:"details,omitempty"` // Used internally to communicate the volume assignment to platform
}

type Outputs struct {
	Artifacts OutputArtifacts `json:"artifacts,omitempty"`
}

type OutputArtifacts map[string]OutputArtifact

type OutputArtifact struct {
	Path          string   `json:"path,omitempty"`
	Excludes      []string `json:"excludes,omitempty"`
	ArchiveMode   string   `json:"archive_mode,omitempty"`
	StorageMethod string   `json:"storage_method,omitempty"`
	From          string   `json:"from,omitempty"`
	Retention     string   `json:"retention,omitempty"`
	MetaData      []string `json:"meta_data,omitempty"`
}

type VolumeRequirements map[string]*Volume

type Volume struct {
	Name         string `json:"name,omitempty"`
	StorageClass string `json:"storage_class,omitempty"`
	SizeGB       string `json:"size_gb,omitempty"`
	// axrn and details are internally used fields for requesting volumes and storing the assignment
	AXRN    string      `json:"axrn,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// TerminationPolicy is the maximum cost or time that a job or deployment should run.
// Although the underlying type are floats, they are unmarshalled as string types, so they can
// be parameterized.
type TerminationPolicy struct {
	SpendingCents string `json:"spending_cents,omitempty"`
	TimeSeconds   string `json:"time_seconds,omitempty"`
}

func (tmpl BaseTemplate) GetType() string {
	return tmpl.Type
}

func (tmpl BaseTemplate) GetVersion() string {
	return string(tmpl.Version)
}

func (tmpl BaseTemplate) GetName() string {
	return tmpl.Name
}

func (tmpl BaseTemplate) GetDescription() string {
	return tmpl.Description
}

func (tmpl BaseTemplate) GetID() string {
	return tmpl.ID
}

func (tmpl BaseTemplate) GetRepo() string {
	return tmpl.Repo
}

func (tmpl BaseTemplate) GetBranch() string {
	return tmpl.Branch
}

func (tmpl BaseTemplate) GetRevision() string {
	return tmpl.Revision
}

func (tmpl BaseTemplate) GetLabels() map[string]string {
	return tmpl.Labels
}

func (tmpl *BaseTemplate) GetInputs() *Inputs {
	return nil
}

func (tmpl *BaseTemplate) GetOutputs() *Outputs {
	return nil
}

func (tmpl BaseTemplate) String() string {
	str := tmpl.GetName()
	info := make([]string, 0)
	if tmpl.ID != "" {
		info = append(info, fmt.Sprintf("ID: %s", tmpl.ID))
	}
	if tmpl.Repo != "" {
		info = append(info, fmt.Sprintf("repo: %s", tmpl.Repo))
	}
	if tmpl.Branch != "" {
		info = append(info, fmt.Sprintf("branch: %s", tmpl.Branch))
	}
	if len(info) > 0 {
		str += fmt.Sprintf(" (%s)", strings.Join(info, ", "))
	}
	return str
}

func (tmpl *BaseTemplate) setRepoInfo(repo, branch, revision string) {
	tmpl.Revision = revision
	tmpl.Repo = repo
	tmpl.Branch = branch
	if repo != "" && branch != "" && tmpl.Name != "" {
		tmpl.ID = GenerateTemplateUUID(repo, branch, tmpl.Name)
	}
}

func (tmpl *BaseTemplate) Validate(preprocess ...bool) *axerror.AXError {
	if tmpl.Name != "" {
		// For inlined containers skip this check
		if tmpl.Version == "" {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'version' required")
		}
		if tmpl.Version != TemplateVersion1 {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Unsupported template version: '%s'. Supported versions: %s", tmpl.Version, TemplateVersion1)
		}
	}
	return nil
}

func (tmpl *BaseTemplate) ValidateContext(context *TemplateBuildContext) *axerror.AXError {
	return nil
}

var (
	invalidParamNameErrStr = "invalid parameter name: '%s'. names must be one word, and contain only alphanumeric, underscore, or dash characters"
)

func (in *Inputs) Validate() *axerror.AXError {
	if in == nil {
		return nil
	}
	for pName := range in.Parameters {
		if !paramNameRegex.MatchString(pName) {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef(invalidParamNameErrStr, pName)
		}
	}
	for pName := range in.Artifacts {
		if !paramNameRegex.MatchString(pName) {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef(invalidParamNameErrStr, pName)
		}
	}
	for pName := range in.Volumes {
		if !paramNameRegex.MatchString(pName) {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef(invalidParamNameErrStr, pName)
		}
	}
	for pName := range in.Fixtures {
		if !paramNameRegex.MatchString(pName) {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef(invalidParamNameErrStr, pName)
		}
	}
	return nil
}

func (in *Inputs) HasInput(argument string) bool {
	if in == nil {
		return false
	}
	parts := strings.SplitN(argument, ".", 2)
	name := parts[1]
	switch parts[0] {
	case InputTypeParameters:
		_, ok := in.Parameters[name]
		return ok
	case InputTypeArtifacts:
		_, ok := in.Artifacts[name]
		return ok
	case InputTypeVolumes:
		_, ok := in.Volumes[name]
		return ok
	case InputTypeFixtures:
		_, ok := in.Fixtures[name]
		return ok
	default:
		return false
	}
}

// parameters returns the input's mapping from %%paramname%% to a param structure containing its name, type, and default value
// It has two use cases:
//   1. build a paramMap of parameter declarations of the current scope
//   2. build a paramMap of input parameters required by a template
// In the first case, we wish to include attributes (such as .path, .mount_path) so they are accessible in the scope.
// In the second case, we only want the inputs ref names to be included in the param map
// includeAttributes indicates if we should also include the sub attributes or not
func (in *Inputs) parameters(includeAttributes bool) paramMap {
	params := make(paramMap)
	for refName, sp := range in.Parameters {
		p := param{
			name:      fmt.Sprintf("inputs.parameters.%s", refName),
			paramType: paramTypeString,
		}
		if sp != nil {
			p.defaultVal = sp.Default
		}
		params[p.name] = p
	}
	for refName := range in.Artifacts {
		p := param{
			name:      fmt.Sprintf("inputs.artifacts.%s", refName),
			paramType: paramTypeArtifact,
		}
		params[p.name] = p
		if includeAttributes {
			p = param{
				name:      fmt.Sprintf("inputs.artifacts.%s.path", refName),
				paramType: paramTypeString,
			}
			params[p.name] = p
		}
	}
	for refName := range in.Volumes {
		p := param{
			name:      fmt.Sprintf("inputs.volumes.%s", refName),
			paramType: paramTypeVolume,
		}
		params[p.name] = p
		if includeAttributes {
			p = param{
				name:      fmt.Sprintf("inputs.volumes.%s.mount_path", refName),
				paramType: paramTypeString,
			}
			params[p.name] = p
		}
	}
	for refName := range in.Fixtures {
		p := param{
			name:      fmt.Sprintf("inputs.fixtures.%s", refName),
			paramType: paramTypeFixture,
		}
		params[p.name] = p
	}
	return params
}

// usedParameters returns any parameter usages from in the inputs within the various sub fields.
// e.g. they are doing something like:
// inputs:
//   parameters:
//     foo:
//       default: "%%other_field%%"
//   artifacts:
//     data:
//       path: "%%mount_path%%"
func (in *Inputs) usedParameters() (paramMap, *axerror.AXError) {
	pMap := make(paramMap)
	if in.Parameters != nil {
		for _, sp := range in.Parameters {
			if sp != nil {
				axErr := pMap.extractUsedParams(sp.Default, paramTypeString)
				if axErr != nil {
					return nil, axErr
				}
			}
		}
	}
	if in.Artifacts != nil {
		for _, inArt := range in.Artifacts {
			if inArt != nil {
				axErr := pMap.extractUsedParams(inArt.From, paramTypeArtifact)
				if axErr != nil {
					return nil, axErr
				}
				axErr = pMap.extractUsedParams(inArt.Path, paramTypeString)
				if axErr != nil {
					return nil, axErr
				}
			}
		}
	}
	if in.Volumes != nil {
		for _, vol := range in.Volumes {
			if vol != nil {
				axErr := pMap.extractUsedParams(vol.From, paramTypeVolume)
				if axErr != nil {
					return nil, axErr
				}
				axErr = pMap.extractUsedParams(vol.MountPath, paramTypeString)
				if axErr != nil {
					return nil, axErr
				}
			}
		}
	}
	// fixtures does not have any subfields to extract
	//if in.Fixtures != nil {
	//}
	return pMap, nil
}

var argNameRegex = regexp.MustCompile("^" + paramNameRegexStr + "\\." + paramNameRegexStr + "$")

// Validate ensure the template field is not empty and arg names are of the form <input_type>.<input_name>
func (tr *TemplateRef) Validate() *axerror.AXError {
	tr.Template = strings.TrimSpace(tr.Template)
	if tr.Template == "" {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'template' field required")
	}
	for argName := range tr.Arguments {
		if !argNameRegex.MatchString(argName) {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("argument '%s' not of the expected format: <input_type>.<input_name> (e.g. 'parameters.%s')", argName, argName)
		}
	}
	return nil
}

// Inlined returns whether or not this an inlined container template, or a reference to a template
func (ic *InlineContainerTemplateRef) Inlined() bool {
	return ic.Template == ""
}

// usedParameters detects what parameters are used in an InlineContainerTemplateRef
func (ic *InlineContainerTemplateRef) usedParameters() (paramMap, *axerror.AXError) {
	pMap := make(paramMap)
	if ic.Inlined() {
		ctParams, axErr := ic.ContainerTemplate.usedParameters()
		if axErr != nil {
			return nil, axErr
		}
		axErr = pMap.merge(ctParams)
		if axErr != nil {
			return nil, axErr
		}
	} else {
		for argName, argVal := range ic.TemplateRef.Arguments {
			if argVal != nil && IsParam(*argVal) {
				// We can determine the type of the parameter because our arguments include type
				// (e.g. artifacts.CODE: "%%steps.STEP1.output.artifacts.CODE%%")
				argNameParts := strings.Split(argName, ".")
				switch argNameParts[0] {
				case InputTypeParameters:
					pMap.extractUsedParams(*argVal, paramTypeString)
				case InputTypeArtifacts:
					pMap.extractUsedParams(*argVal, paramTypeArtifact)
				case InputTypeVolumes:
					pMap.extractUsedParams(*argVal, paramTypeVolume)
				case InputTypeFixtures:
					pMap.extractUsedParams(*argVal, paramTypeFixture)
				default:
					pMap.extractUsedParams(*argVal, paramTypeString)
				}
			}
		}
	}
	return pMap, nil
}

// UnmarshalJSON is a custom unmarshaller specific to InlineContainerTemplateRef in order to ensure that the
// user supplied either a template reference, or an inlined container, but not both.
func (ic *InlineContainerTemplateRef) UnmarshalJSON(b []byte) error {
	var jsonMap map[string]interface{}
	err := json.Unmarshal(b, &jsonMap)
	if err != nil {
		return err
	}
	_, isTemplateRef := jsonMap["template"]

	// mapstructure is used ensure the user does not supply container fields when using a template reference
	// and vice versa. We decode to the opposite template ref or container template, and then see if there were
	// anyused fields. If so, it means the user mixed template reference with container fields
	if isTemplateRef {
		var md mapstructure.Metadata
		var ctr ContainerTemplate
		config := &mapstructure.DecoderConfig{
			Result:   &ctr,
			Metadata: &md,
			TagName:  "json",
		}
		if decoder, err := mapstructure.NewDecoder(config); err == nil {
			if err = decoder.Decode(jsonMap); err == nil {
				for _, key := range md.Keys {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("cannot have both 'template' field and container field '%s'", key)
				}
			}
		}
		var tmplRef TemplateRef
		err := json.Unmarshal(b, &tmplRef)
		if err != nil {
			return err
		}
		*ic = InlineContainerTemplateRef{
			tmplRef,
			ContainerTemplate{},
		}
	} else {
		var md mapstructure.Metadata
		var t TemplateRef
		config := &mapstructure.DecoderConfig{
			Result:   &t,
			Metadata: &md,
			TagName:  "json",
		}
		if decoder, err := mapstructure.NewDecoder(config); err == nil {
			if err = decoder.Decode(jsonMap); err == nil {
				for _, key := range md.Keys {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("cannot mix container fields with template reference field '%s'", key)
				}
			}
		}
		var ctr ContainerTemplate
		err := json.Unmarshal(b, &ctr)
		if err != nil {
			return err
		}
		*ic = InlineContainerTemplateRef{
			TemplateRef{},
			ctr,
		}
	}
	return nil
}

// Validate calls either the TemplateRef's or ContainerTemplate's Validate() method depending if it is inlined or not
func (ic *InlineContainerTemplateRef) Validate() *axerror.AXError {
	if ic.Inlined() {
		axErr := ic.ContainerTemplate.Validate()
		if axErr != nil {
			return axErr
		}
	} else {
		axErr := ic.TemplateRef.Validate()
		if axErr != nil {
			return axErr
		}
	}
	return nil
}

// Validate ensures volume reference name is acceptable by k8s and calls validate against each volume
func (vols VolumeRequirements) Validate() *axerror.AXError {
	for refName, vol := range vols {
		if !common.ValidateKubeObjName(refName) {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Invalid volume reference name '%s'. Expected format ^([a-z0-9]([-a-z0-9]*[a-z0-9])?)$", refName)
		}
		axErr := vol.Validate()
		if axErr != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("volumes.%v: %v", refName, axErr)
		}
	}
	return nil
}

// Validate ensures that the volume is either anonymous or named but not both
func (vol *Volume) Validate() *axerror.AXError {
	if vol.Name != "" {
		if vol.StorageClass != "" || vol.SizeGB != "" {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'%s' should not supply 'storage_class' or 'size_gb'", vol.Name)
		}
	} else if vol.StorageClass != "" {
		if vol.SizeGB == "" {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("'size_gb' is required for anonymous volume requests")
		}
	} else {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("requirement should either supply a 'name' or 'storage_class'")
	}
	return nil
}

func (vol *Volume) Equals(o Volume) bool {
	return vol.StorageClass == vol.StorageClass && vol.SizeGB == o.SizeGB && vol.Name == o.Name
}

func GenerateTemplateUUID(repo, branch, name string) string {
	return common.GenerateUUIDv5(fmt.Sprintf("%s:%s:%s", repo, branch, name))
}
