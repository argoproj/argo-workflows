package template

import "applatix.io/axerror"

// NOTE: squash is a mapstructure struct tag but we teach mapstructure to parse the json tags
type ProjectTemplate struct {
	BaseTemplate `json:",squash"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Categories   []string               `json:"categories,omitempty"`
	Assets       *ProjectAssets         `json:"assets,omitempty"`
	Actions      map[string]TemplateRef `json:"actions,omitempty"`
	Publish      *ProjectPublish        `json:"publish,omitempty"`
}

type ProjectPublish struct {
	Branches []string `json:"branches,omitempty"`
}

type ProjectAssets struct {
	Icon          string `json:"icon,omitempty"`
	Detail        string `json:"detail,omitempty"`
	PublisherIcon string `json:"publisher_icon,omitempty"`
}

func (tmpl *ProjectTemplate) GetInputs() *Inputs {
	return nil
}

func (tmpl *ProjectTemplate) GetOutputs() *Outputs {
	return nil
}

func (tmpl *ProjectTemplate) Validate(preproc ...bool) *axerror.AXError {
	if axErr := tmpl.BaseTemplate.Validate(); axErr != nil {
		return axErr
	}
	if tmpl.Categories == nil || len(tmpl.Categories) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Project has no categories")
	}
	if tmpl.Actions == nil || len(tmpl.Actions) == 0 {
		return axerror.ERR_API_INVALID_PARAM.NewWithMessage("Project has no actions")
	}
	//tmpl.Categories = utils.DedupStringList(p.Categories)
	return nil
}

func (tmpl *ProjectTemplate) ValidateContext(context *TemplateBuildContext) *axerror.AXError {
	emptyScope := make(paramMap)
	for actionName, action := range tmpl.Actions {
		if action.Template == "" {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s 'template' required", actionName)
		}
		res, exists := context.Results[action.Template]
		if !exists {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s template '%s' does not exist", actionName, action.Template)
		}
		if res.AXErr != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s template '%s' is not valid", actionName, action.Template)
		}
		switch res.Template.GetType() {
		case TemplateTypeContainer, TemplateTypeWorkflow, TemplateTypeDeployment:
			break
		default:
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s.template %s must be of type container, workflow, or deployment", actionName, action.Template)
		}
		unresolved, axErr := validateReceiverParamsPartial(res.Template.GetName(), res.Template.GetInputs(), action.Arguments, emptyScope)
		if axErr != nil {
			return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s: %v", actionName, axErr)
		}
		// Need to ensure that all the unresolved params are parameter types, not volumes/artifacts/etc...
		// We have no facility to send volumes/artifacts as inputs to a top level job
		for _, param := range unresolved {
			if param.paramType != paramTypeString {
				return axerror.ERR_API_INVALID_PARAM.NewWithMessagef("actions.%s: template '%s' requires input '%s' of type: %s, which is disallowed", actionName, action.Template, param.name, param.paramType)
			}
		}
	}
	return nil
}
