package service

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"applatix.io/axerror"
	"applatix.io/template"
)

type EmbeddedTemplateIf interface {
	String() string
	GetID() string
	GetRepo() string
	GetBranch() string
	GetRevision() string
	GetName() string
	GetDescription() string
	GetType() string
	GetVersion() string
	GetLabels() map[string]string
	GetInputs() *template.Inputs
	GetOutputs() *template.Outputs
	SetStats(cost *float64, jobsFail *int64, jobsSuccess *int64)
	GetStats() *TemplateStats
	SubstituteArguments(template.Arguments) (EmbeddedTemplateIf, *axerror.AXError)
}

type TemplateStats struct {
	Cost        *float64 `json:"cost,omitempty"`
	JobsFail    *int64   `json:"jobs_fail,omitempty"`
	JobsSuccess *int64   `json:"jobs_success,omitempty"`
}

type EmbeddedContainerTemplate struct {
	*template.ContainerTemplate
	TemplateStats
}

// EmbeddedDeploymentTemplate is the same as a deployment template, but the
// templates in the containers section, are fully embedded.
type EmbeddedDeploymentTemplate struct {
	*template.DeploymentTemplate
	Containers map[string]*Service `json:"containers,omitempty"`
	TemplateStats
}

// EmbeddedWorkflowTemplate is the same as a WorkflowTemplate, but any template
// references in the steps and fixtures section, are fully embedded.
type EmbeddedWorkflowTemplate struct {
	*template.WorkflowTemplate
	Steps    []map[string]*Service                    `json:"steps,omitempty"`
	Fixtures []map[string]*EmbeddedFixtureTemplateRef `json:"fixtures,omitempty"`
	TemplateStats
}

// EmbeddedContainerTemplateRef is a template ref that can only refer to containers
type EmbeddedContainerTemplateRef struct {
	Template *EmbeddedContainerTemplate `json:"template,omitempty"`
	*Service
}

// EmbeddedFixtureTemplateRef is the embedded version of FixtureTemplateRef
// which can either refer to embedded dynamic fixture, or regular fixture requirements
type EmbeddedFixtureTemplateRef struct {
	*EmbeddedContainerTemplateRef
	*template.FixtureRequirement
}

// EmbedServiceTemplate accepts a template (derived from YAML) and returns the fully embedded version of it
// which is suitable to be stored in service objects and the templates table. It expands any syntatic sugar
// used in the YAML (such as template inlining).
func EmbedServiceTemplate(tmpl template.TemplateIf, ctx *template.TemplateBuildContext) (EmbeddedTemplateIf, *axerror.AXError) {
	switch tmpl.GetType() {
	case template.TemplateTypeContainer:
		ct := tmpl.(*template.ContainerTemplate)
		return EmbedContainerTemplate(ct, ctx)
	case template.TemplateTypeDeployment:
		dt := tmpl.(*template.DeploymentTemplate)
		return EmbedDeploymentTemplate(dt, ctx)
	case template.TemplateTypeWorkflow:
		wt := tmpl.(*template.WorkflowTemplate)
		return EmbedWorkflowTemplate(wt, ctx)
	}
	return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("%s templates cannot be embedded", tmpl.GetType())
}

func EmbedContainerTemplate(tmpl *template.ContainerTemplate, ctx *template.TemplateBuildContext) (*EmbeddedContainerTemplate, *axerror.AXError) {
	b, err := json.Marshal(tmpl)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(err.Error())
	}
	var copyTmpl EmbeddedContainerTemplate
	err = json.Unmarshal(b, &copyTmpl)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(err.Error())
	}
	return &copyTmpl, nil
}

func EmbedDeploymentTemplate(tmpl *template.DeploymentTemplate, ctx *template.TemplateBuildContext) (*EmbeddedDeploymentTemplate, *axerror.AXError) {
	b, err := json.Marshal(tmpl)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(err.Error())
	}
	var copyTmpl template.DeploymentTemplate
	err = json.Unmarshal(b, &copyTmpl)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(err.Error())
	}
	eTmpl := EmbeddedDeploymentTemplate{
		DeploymentTemplate: &copyTmpl,
	}
	eTmpl.Containers = make(map[string]*Service)
	for refName, icr := range tmpl.Containers {
		ectr := Service{}
		if icr.Inlined() {
			ctrTmpl, arguments, axErr := icr.ReverseInline(fmt.Sprintf("%s.%s", tmpl.Name, refName))
			if axErr != nil {
				return nil, axErr
			}
			eCtrTmpl, axErr := EmbedContainerTemplate(ctrTmpl, ctx)
			if axErr != nil {
				return nil, axErr
			}
			ectr.Template = eCtrTmpl
			ectr.Arguments = arguments
		} else {
			eTmplIf, axErr := EmbedServiceTemplate(ctx.Templates[icr.Template], ctx)
			if axErr != nil {
				return nil, axErr
			}
			ectr.Template = eTmplIf.(*EmbeddedContainerTemplate)
			if icr.Arguments != nil {
				ectr.Arguments = icr.Arguments
			} else {
				ectr.Arguments = make(template.Arguments)
			}
			axErr = inferArgumentsToChild(tmpl, ctx.Templates[icr.Template], ectr.Arguments)
			if axErr != nil {
				return nil, axErr
			}
		}
		eTmpl.Containers[refName] = &ectr
	}
	// NOTE: we do not have to worry about embedding fixture templates for deployments,
	// since deployments cannot have dynamic fixtures -- we verify this in Validate().
	return &eTmpl, nil
}

func EmbedWorkflowTemplate(tmpl *template.WorkflowTemplate, ctx *template.TemplateBuildContext) (eTmpl *EmbeddedWorkflowTemplate, axErr *axerror.AXError) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Failed to embed workflow template: %s:\n%s", r, debug.Stack())
			axErr = axerror.ERR_AX_INTERNAL.NewWithMessagef("%v", r)
		}
	}()
	b, err := json.Marshal(tmpl)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(err.Error())
	}
	var copyTmpl template.WorkflowTemplate
	err = json.Unmarshal(b, &copyTmpl)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage(err.Error())
	}
	eTmpl = &EmbeddedWorkflowTemplate{
		WorkflowTemplate: &copyTmpl,
	}
	eTmpl.Steps = make([]map[string]*Service, len(tmpl.Steps))
	for i, stepMap := range tmpl.Steps {
		eTmpl.Steps[i] = make(map[string]*Service)
		for stepName, wfStep := range stepMap {
			ewfs := Service{}
			if wfStep.Inlined() {
				ctrTmpl, arguments, axErr := wfStep.ReverseInline(fmt.Sprintf("%s.%s", tmpl.Name, stepName))
				if axErr != nil {
					return nil, axErr
				}
				eCtrTmpl, axErr := EmbedContainerTemplate(ctrTmpl, ctx)
				if axErr != nil {
					return nil, axErr
				}
				ewfs.Template = eCtrTmpl
				ewfs.Arguments = arguments
			} else {
				// This is recursive
				childTmpl, axErr := EmbedServiceTemplate(ctx.Templates[wfStep.Template], ctx)
				if axErr != nil {
					return nil, axErr
				}
				ewfs.Template = childTmpl
				if wfStep.Arguments != nil {
					ewfs.Arguments = wfStep.Arguments
				} else {
					ewfs.Arguments = make(template.Arguments)
				}
				axErr = inferArgumentsToChild(tmpl, ctx.Templates[wfStep.Template], ewfs.Arguments)
				if axErr != nil {
					return nil, axErr
				}
			}
			// Our YAML accepts comma separated list. The embedded version
			// wants a map converting from YAML to embedded,
			if wfStep.Flags != "" {
				ewfs.Flags = make(map[string]interface{})
				for _, flag := range strings.Split(wfStep.Flags, ",") {
					flag = strings.TrimSpace(flag)
					ewfs.Flags[flag] = true
				}
			}
			eTmpl.Steps[i][stepName] = &ewfs

		}
	}
	if tmpl.Fixtures != nil && len(tmpl.Fixtures) > 0 {
		eTmpl.Fixtures = make([]map[string]*EmbeddedFixtureTemplateRef, len(tmpl.Fixtures))
		for i, fixReqMap := range tmpl.Fixtures {
			eTmpl.Fixtures[i] = make(map[string]*EmbeddedFixtureTemplateRef)
			for fixRefName, fixTmplRef := range fixReqMap {
				if fixTmplRef.IsDynamicFixture() {
					eCtrTmpl, axErr := EmbedServiceTemplate(ctx.Templates[fixTmplRef.Template], ctx)
					if axErr != nil {
						return nil, axErr
					}
					cRef := EmbeddedContainerTemplateRef{
						Template: eCtrTmpl.(*EmbeddedContainerTemplate),
						Service:  &Service{},
					}
					cRef.Arguments = fixTmplRef.Arguments
					eTmpl.Fixtures[i][fixRefName] = &EmbeddedFixtureTemplateRef{
						EmbeddedContainerTemplateRef: &cRef,
					}
				} else {
					eTmpl.Fixtures[i][fixRefName] = &EmbeddedFixtureTemplateRef{
						FixtureRequirement: &fixTmplRef.FixtureRequirement,
					}
				}
			}
		}
	}
	return eTmpl, nil
}

// UnmarshalEmbeddedTemplate unmarshalls an embedded template and returns the EmbeddedTemplateIf
func UnmarshalEmbeddedTemplate(data []byte) (EmbeddedTemplateIf, *axerror.AXError) {
	var tmpl template.BaseTemplate
	axErr := template.Unmarshal(data, &tmpl, false)
	if axErr != nil {
		return nil, axErr
	}
	switch tmpl.Type {
	case template.TemplateTypeContainer:
		var tmpl EmbeddedContainerTemplate
		err := json.Unmarshal(data, &tmpl)
		if err != nil {
			return &tmpl, axerror.ERR_API_INVALID_PARAM.NewWithMessagef(err.Error())
		}
		return &tmpl, nil
	case template.TemplateTypeWorkflow:
		tmpl := EmbeddedWorkflowTemplate{WorkflowTemplate: &template.WorkflowTemplate{}}
		err := json.Unmarshal(data, &tmpl)
		if err != nil {
			return &tmpl, axerror.ERR_API_INVALID_PARAM.NewWithMessagef(err.Error())
		}
		return &tmpl, nil
	case template.TemplateTypeDeployment:
		tmpl := EmbeddedDeploymentTemplate{DeploymentTemplate: &template.DeploymentTemplate{}}
		err := json.Unmarshal(data, &tmpl)
		if err != nil {
			return &tmpl, axerror.ERR_API_INVALID_PARAM.NewWithMessagef(err.Error())
		}
		return &tmpl, nil
	default:
		return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Unsupported template type: '%s'", tmpl.Type)
	}
}

// UnmarshalJSON is the custom unmarshal method for a workflow template.
// EmbeddedWorkflowTemplate needs to unmarshal steps and fixtures separately
// from the embedded WorkflowTemplate, which have a different structure
func (tmpl *EmbeddedWorkflowTemplate) UnmarshalJSON(b []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Failed to unmarshal workflow template: %s:\n%s", r, debug.Stack())
			err = fmt.Errorf("%v", r)
		}
	}()
	var jsonMap map[string]*json.RawMessage
	err = json.Unmarshal(b, &jsonMap)
	if err != nil {
		return err
	}
	// First unmarshall steps and fixtures independently
	if _, ok := jsonMap["steps"]; ok {
		err = json.Unmarshal([]byte(*jsonMap["steps"]), &tmpl.Steps)
		if err != nil {
			return err
		}
		if len(tmpl.Steps) == 0 {
			panic("empty step list")
		}
		delete(jsonMap, "steps")
	}

	if _, ok := jsonMap["fixtures"]; ok {
		err = json.Unmarshal([]byte(*jsonMap["fixtures"]), &tmpl.Fixtures)
		if err != nil {
			return err
		}
		if len(tmpl.Fixtures) == 0 {
			panic("empty fixture list")
		}
		delete(jsonMap, "fixtures")
	}

	// marshal back the jsonMap back to bytes (minus steps/fixtures),
	// then unmarshal the bytes (there's probably a better way to do this)
	b, err = json.Marshal(jsonMap)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &tmpl.WorkflowTemplate)
	if err != nil {
		return err
	}
	return err
}

func (tmpl *EmbeddedFixtureTemplateRef) UnmarshalJSON(b []byte) error {
	var jsonMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &jsonMap)
	if err != nil {
		return err
	}
	if _, ok := jsonMap["template"]; ok {
		err = json.Unmarshal(b, &tmpl.EmbeddedContainerTemplateRef)
	} else {
		err = json.Unmarshal(b, &tmpl.FixtureRequirement)
	}
	return err
}

func (tmpl *EmbeddedFixtureTemplateRef) IsDynamicFixture() bool {
	return tmpl.EmbeddedContainerTemplateRef != nil
}

func (tmpl *EmbeddedContainerTemplateRef) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &tmpl.Service)
	if err != nil {
		return err
	}
	if tmpl.Service.Template != nil {
		tmpl.Template = tmpl.Service.Template.(*EmbeddedContainerTemplate)
	}
	return nil
}

// inferArgumentsToChild supports the case where a user omits some 'arguments' fields when
// calling a child template in a step. In this case, we map the arguments from the inputs
// of the parent to the child. Special consideration should be made for default values
// in the child, as these are not strictly required to be in the parents inputs.
func inferArgumentsToChild(parent, child template.TemplateIf, args template.Arguments) *axerror.AXError {
	for argName := range template.GetTemplateArguments(child) {
		if _, ok := args[argName]; !ok {
			// add a entry to the arguments. use nil as a placeholder nil, to indicate
			// the argument needs to be satsified by caller
			args[argName] = nil
		}
	}
	parentInputs := parent.GetInputs()
	for argName, argVal := range args {
		if argVal != nil {
			// parent explicitly sent a value to child
			continue
		}
		if parentInputs.HasInput(argName) {
			// the parent has an input, exactly the same as child. we pass the same value to child
			argVal := fmt.Sprintf("%%%%inputs.%s%%%%", argName)
			args[argName] = &argVal
			continue
		}
		// The child argument was not satisfied by parent's input. The following checks
		// to see if there is a default value in child
		if strings.HasPrefix(argName, "parameters.") {
			name := strings.Split(argName, ".")[1]
			param := child.GetInputs().Parameters[name]
			if param != nil && param.Default != nil {
				// There is a default value in the child. Caller does not
				// need to pass in any arguments. Delete the arg from the map.
				delete(args, argName)
				continue
			}
		}
		return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("argument '%s' to child template '%s' was not satisfied", argName, child.GetName())
	}
	return nil
}

func (ts *TemplateStats) SetStats(cost *float64, jobsFail *int64, jobsSuccess *int64) {
	if cost != nil {
		ts.Cost = cost
	}
	if jobsFail != nil {
		ts.JobsFail = jobsFail
	}
	if jobsSuccess != nil {
		ts.JobsSuccess = jobsSuccess
	}
}

func (ts *TemplateStats) GetStats() *TemplateStats {
	return ts
}
