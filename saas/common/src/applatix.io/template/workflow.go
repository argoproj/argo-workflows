package template

import (
	"fmt"
	"strings"

	"applatix.io/axerror"
)

// Available flags to use in a workflow step
const (
	WorkflowStepFlagIgnoreError = "ignore_error"
	WorkflowStepFlagAutoRetry   = "auto_retry"
	WorkflowStepFlagAlwaysRun   = "always_run"
	WorkflowStepFlagSkipped     = "skipped"
)

var workflowStepFlags = []string{WorkflowStepFlagIgnoreError, WorkflowStepFlagAutoRetry, WorkflowStepFlagAlwaysRun, WorkflowStepFlagSkipped}
var workflowStepFlagMap = map[string]bool{}

func init() {
	for _, flagName := range workflowStepFlags {
		workflowStepFlagMap[flagName] = true
	}
}

// NOTE: squash is a mapstructure struct tag but we teach mapstructure to parse the json tags
type WorkflowTemplate struct {
	BaseTemplate      `json:",squash"`
	Inputs            *Inputs                   `json:"inputs,omitempty"`
	Outputs           *Outputs                  `json:"outputs,omitempty"`
	Steps             []map[string]WorkflowStep `json:"steps,omitempty"`
	Fixtures          FixtureRequirements       `json:"fixtures,omitempty"`
	Volumes           VolumeRequirements        `json:"volumes,omitempty"`
	ArtifactTags      []string                  `json:"artifact_tags,omitempty"`
	TerminationPolicy *TerminationPolicy        `json:"termination_policy,omitempty"`
	//Annotations       map[string]string         `json:"annotations,omitempty"` - I think this is a service concept, not a template
}

// WorkflowStep is either a template ref, an inlined container, with added flags
type WorkflowStep struct {
	InlineContainerTemplateRef `json:",squash"`
	Flags                      string `json:"flags,omitempty"`
	//Flags                      map[string]bool `json:"flags,omitempty"`
}

func (tmpl *WorkflowTemplate) GetInputs() *Inputs {
	return tmpl.Inputs
}

func (tmpl *WorkflowTemplate) GetOutputs() *Outputs {
	return tmpl.Outputs
}

func (tmpl *WorkflowTemplate) Validate(preproc ...bool) *axerror.AXError {
	if axErr := tmpl.BaseTemplate.Validate(); axErr != nil {
		return axErr
	}
	if axErr := tmpl.Inputs.Validate(false); axErr != nil {
		return axErr
	}
	if axErr := tmpl.Fixtures.Validate(); axErr != nil {
		return axErr
	}
	// used to detect duplicate step names
	stepRefNameMap := map[string]WorkflowStep{}
	for i, parallelSteps := range tmpl.Steps {
		if len(parallelSteps) > 1 && tmpl.Volumes != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("steps[%d]: workflows with volumes cannot have parallel steps", i)
		}
		for stepRefName, wfStep := range parallelSteps {
			if !paramNameRegex.MatchString(stepRefName) {
				return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("steps[%d].%s: invalid step name. names must be one word, and contain only alphanumeric, underscore, or dash characters", i, stepRefName)
			}
			if _, ok := stepRefNameMap[stepRefName]; ok {
				return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("steps[%d].%s: duplicated step name", i, stepRefName)
			}
			stepRefNameMap[stepRefName] = wfStep

			axErr := wfStep.Validate()
			if axErr != nil {
				return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("steps[%d].%s: %v", i, stepRefName, axErr)
			}
			if !wfStep.Inlined() {
				for _, val := range wfStep.Arguments {
					if val != nil && listExpansionParamRegex.MatchString(*val) {
						return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("steps[%d].%s: workflows with volumes cannot have parallel steps", i, stepRefName)
					}
				}
			} else {
				_, _, axErr = wfStep.ReverseInline("temp")
				if axErr != nil {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("steps[%d].%s: %v", i, stepRefName, axErr)
				}
			}
		}
	}

	if tmpl.Volumes != nil {
		axErr := tmpl.Volumes.Validate()
		if axErr != nil {
			return axErr
		}
	}

	return nil
}

func (tmpl *WorkflowTemplate) parametersInScope() paramMap {
	return getParameterDeclarations(tmpl.Inputs, tmpl.Volumes, tmpl.Fixtures)
}

// usedParameters detects all the parameters used in various parts of a workflow template and returns it in a paramMap.
// For workflows, check inline container's command, args, env, image
func (tmpl *WorkflowTemplate) usedParameters() (paramMap, *axerror.AXError) {
	pMap := make(paramMap)
	axErr := pMap.extractUsedParams(tmpl.Labels, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.ArtifactTags, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	// axErr = pMap.extractUsedParams(tmpl.Annotations, paramTypeString)
	// if axErr != nil {
	// 	return nil, axErr
	// }
	axErr = pMap.extractUsedParams(tmpl.Fixtures, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.Volumes, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	axErr = pMap.extractUsedParams(tmpl.TerminationPolicy, paramTypeString)
	if axErr != nil {
		return nil, axErr
	}
	// merge any used parameters workflow steps
	for i, parallelSteps := range tmpl.Steps {
		for stepRefName, wfStep := range parallelSteps {
			ctParams, axErr := wfStep.usedParameters()
			if axErr != nil {
				return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("steps[%d].%s: %v", i, stepRefName, axErr)
			}
			axErr = pMap.merge(ctParams)
			if axErr != nil {
				return nil, axerror.ERR_API_INVALID_PARAM.NewWithMessagef("steps[%d].%s: %v", i, stepRefName, axErr)
			}
		}
	}
	return pMap, nil
}

func (tmpl *WorkflowTemplate) ValidateContext(context *TemplateBuildContext) *axerror.AXError {
	// Check if we already processed this template and return previous result
	if context.Processed(tmpl.Name) {
		return context.Results[tmpl.Name].AXErr
	}
	// Check if we are in an infinite recursive workflow
	if context.depth > 50 {
		return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("infinite recursive workflow")
	}

	// Start out with scoped parameters from inputs, fixtures, volumes.
	// As we process sequential groups of steps, the subsequent steps can reference outputs from previous steps
	// (e.g. a build step will reference %%steps.checkout.code%%)
	// As we complete validation of each sequential group of steps, we continuously add any output artifacts from
	// the previous step group to the current scope
	scopedParams := tmpl.parametersInScope()

	// dynFixtureOutputs keeps track of any outputs produced by dynamic fixtures.
	// We will add this to the scope after all steps complete
	dynFixtureOutputs := paramMap{}

	// Verify any dynamic fixtures. Template references must be container type
	for i, parallelFixtures := range tmpl.Fixtures {
		for fixRefName, ftr := range parallelFixtures {
			if !ftr.IsDynamicFixture() {
				// Managed fixtures have no context to validate
				continue
			}
			st, ok := context.Templates[ftr.Template]
			if !ok {
				return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("fixtures[%d].%s.template: template '%s' does not exist", i, fixRefName, ftr.Template)
			}
			if st.GetType() != TemplateTypeContainer {
				return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("fitxures[%d].%s.template: '%s' is not a container template", i, fixRefName, ftr.Template)
			}
			ctrTemplate := st.(*ContainerTemplate)
			axErr := validateReceiverParams(st.GetName(), ctrTemplate.Inputs, ftr.Arguments, scopedParams)
			if axErr != nil {
				return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("fixtures[%d].%s: %v", i, fixRefName, axErr)
			}
			if ctrTemplate.Outputs != nil {
				for artRefName := range ctrTemplate.Outputs.Artifacts {
					p := param{
						name:      fmt.Sprintf("fixtures.%s.outputs.artifacts.%s", fixRefName, artRefName),
						paramType: paramTypeArtifact,
					}
					dynFixtureOutputs[p.name] = p
				}
			}
		}
	}

	// The following loop iterates each step, and adds any of the steps output artifacts to the scoped parameters.
	for i, parallelSteps := range tmpl.Steps {
		// stepArtifactParams holds any output params (e.g. %%step.checkout.code%%) until we finish processing all parallel steps in this step group.
		// Once all parallel steps have been processed in this step group, we will add any of their outputs to the parameter scope, so that they are
		// accessible in the next sequential step group.
		stepArtifactParams := paramMap{}

		for stepRefName, wfStep := range parallelSteps {
			var stepOutputs *Outputs

			if wfStep.Inlined() {
				ctParams, axErr := wfStep.usedParameters()
				if axErr != nil {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("steps[%d].%s: %v", i, stepRefName, axErr)
				}
				axErr = validateParams(scopedParams, ctParams)
				if axErr != nil {
					return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("steps[%d].%s: %v", i, stepRefName, axErr)
				}
				stepOutputs = wfStep.Outputs
			} else {
				st, ok := context.Templates[wfStep.Template]
				if !ok {
					return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("steps[%d].%s.template: template '%s' does not exist", i, stepRefName, wfStep.Template)
				}

				switch st.GetType() {
				case TemplateTypeContainer, TemplateTypeDeployment:
					break
				case TemplateTypeWorkflow:
					// Unlike deployments and containers, which should have already been processed, with workflows,
					// we may come across a child workflow that has yet to been processed. The following will
					// recursively process any child workflows, before proceeding with validation of the current one.
					wfTemplate := st.(*WorkflowTemplate)
					if !context.Processed(wfStep.Template) {
						context.depth++
						axErr := wfTemplate.ValidateContext(context)
						context.depth--
						context.MarkProcessed(st, axErr)
					}
				default:
					return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("steps[%d].%s: template '%s' must be of type: container, workflow, template", i, stepRefName, wfStep.Template)
				}

				// At this point, the child template is already processed. check if it is valid
				result := context.Results[wfStep.Template]
				if result.AXErr != nil {
					return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("steps[%d].%s: child template '%s' is not valid", i, stepRefName, wfStep.Template)
				}

				// Ensure we can satisfy all inputs in the reciever template
				axErr := validateReceiverParams(st.GetName(), st.GetInputs(), wfStep.Arguments, scopedParams)
				if axErr != nil {
					return axerror.ERR_AXDB_INVALID_PARAM.NewWithMessagef("steps[%d].%s: %v", i, stepRefName, axErr)
				}
				stepOutputs = st.GetOutputs()
			}

			if stepOutputs != nil {
				// put this step's outputs into the step-level scope holding area,
				// which we will later add to the scope once all parallel steps have completed.
				for artRefName := range stepOutputs.Artifacts {
					p := param{
						name:      fmt.Sprintf("steps.%s.outputs.artifacts.%s", stepRefName, artRefName),
						paramType: paramTypeArtifact,
					}
					stepArtifactParams[p.name] = p
				}
			}
		}
		// all parallel steps succesful. we can now add any output artifacts to the scope
		// and continue with the next sequential step.
		axErr := scopedParams.merge(stepArtifactParams)
		if axErr != nil {
			return axErr
		}
	}

	// Add dynamic fixtures outputs to the scope
	axErr := scopedParams.merge(dynFixtureOutputs)
	if axErr != nil {
		return axErr
	}

	// Do one last validation of all parameters used in the template
	usedParams, axErr := tmpl.usedParameters()
	if axErr != nil {
		return axErr
	}
	axErr = validateParams(scopedParams, usedParams)
	if axErr != nil {
		return axErr
	}

	// See if this workflow is exporting any artifacts. If so, verify that 'from' is
	// referencing a valid step or fixture, and that step has expected artifact with the name.
	if tmpl.Outputs != nil && tmpl.Outputs.Artifacts != nil {
		for refName, art := range tmpl.Outputs.Artifacts {
			if art.From == "" {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("outputs.artifacts.%s 'from' field is required", refName)
			}
			if art.Path != "" {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("outputs.artifacts.%s 'path' field is only valid in container templates, not workflow", refName)
			}
			if !ouputArtifactRegexp.MatchString(art.From) {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("outputs.artifacts.%s.from invalid format '%s'. expected format: '%%%%(steps|fixtures).<REF_NAME>.outputs.artifacts.<ARTIFACT_NAME>%%%%'", refName, art.From)
			}
			pName := strings.Trim(art.From, "%")
			_, ok := scopedParams[pName]
			if !ok {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("outputs.artifacts.%s.from: no output artifact '%s'", refName, pName)
			}
		}
	}

	return nil
}

func (wfs *WorkflowStep) Validate() *axerror.AXError {
	axErr := wfs.InlineContainerTemplateRef.Validate()
	if axErr != nil {
		return axErr
	}
	if wfs.Flags != "" {
		flagsList := strings.Split(wfs.Flags, ",")
		for _, flag := range flagsList {
			flag = strings.TrimSpace(flag)
			if _, valid := workflowStepFlagMap[flag]; !valid {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("unknown flag: %s", flag)
			}
		}
	}
	return nil
}
