package common

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/argoproj/argo/errors"
	"github.com/valyala/fasttemplate"
)

// wfValidationCtx is the context for validating a workflow spec
type wfValidationCtx struct {
	wf      *wfv1.Workflow
	results map[string]validationResult
}

type validationResult struct {
	outputs *wfv1.Outputs
}

func ValidateWorkflow(wf *wfv1.Workflow) error {
	err := VerifyUniqueNonEmptyNames(wf.Spec.Templates)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates%s", err.Error())
	}

	ctx := wfValidationCtx{
		wf:      wf,
		results: make(map[string]validationResult),
	}
	if ctx.wf.Spec.Entrypoint == "" {
		return errors.New(errors.CodeBadRequest, "spec.entrypoint is required")
	}
	entryTmpl := ctx.wf.GetTemplate(ctx.wf.Spec.Entrypoint)
	if entryTmpl == nil {
		return errors.Errorf(errors.CodeBadRequest, "spec.entrypoint template '%s' undefined", ctx.wf.Spec.Entrypoint)
	}

	return ctx.validateTemplate(entryTmpl, ctx.wf.Spec.Arguments, ctx.wf.Spec.Arguments.Parameters)
}

func (ctx *wfValidationCtx) validateTemplate(tmpl *wfv1.Template, args wfv1.Arguments, wfGlobalParameters []wfv1.Parameter) error {
	_, ok := ctx.results[tmpl.Name]
	if ok {
		// we already processed this template
		return nil
	}
	if tmpl.Name == "" {
		errors.Errorf(errors.CodeBadRequest, "template names are required")
	}
	ctx.results[tmpl.Name] = validationResult{}
	_, err := ProcessArgs(tmpl, args, wfGlobalParameters, true)
	if err != nil {
		return err
	}
	scope, err := validateInputs(tmpl)
	if err != nil {
		return err
	}
	err = validateWFGlobalParams(tmpl, wfGlobalParameters, scope)
	if err != nil {
		return err
	}
	if tmpl.Steps == nil {
		err = validateLeaf(scope, tmpl)
	} else {
		err = ctx.validateSteps(scope, tmpl)
	}
	if err != nil {
		return err
	}

	err = validateOutputs(tmpl)
	if err != nil {
		return err
	}
	return nil
}

func validateInputs(tmpl *wfv1.Template) (map[string]interface{}, error) {
	err := VerifyUniqueNonEmptyNames(tmpl.Inputs.Parameters)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "template '%s' inputs.parameters%s", tmpl.Name, err.Error())
	}
	err = VerifyUniqueNonEmptyNames(tmpl.Inputs.Artifacts)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "template '%s' inputs.artifacts%s", tmpl.Name, err.Error())
	}
	scope := make(map[string]interface{})
	for _, param := range tmpl.Inputs.Parameters {
		scope[fmt.Sprintf("inputs.parameters.%s", param.Name)] = true
	}
	isLeaf := tmpl.Container != nil || tmpl.Script != nil
	for _, art := range tmpl.Inputs.Artifacts {
		artRef := fmt.Sprintf("inputs.artifacts.%s", art.Name)
		scope[artRef] = true
		if isLeaf {
			if art.Path == "" {
				return nil, errors.Errorf(errors.CodeBadRequest, "template '%s' %s.path not specified", tmpl.Name, artRef)
			}
		} else {
			if art.Path != "" {
				return nil, errors.Errorf(errors.CodeBadRequest, "template '%s' %s.path only valid in container/script templates", tmpl.Name, artRef)
			}
		}
		if art.From != "" {
			return nil, errors.Errorf(errors.CodeBadRequest, "template '%s' %s.from only valid in arguments", tmpl.Name, artRef)
		}
		errPrefix := fmt.Sprintf("template '%s' %s", tmpl.Name, artRef)
		err = validateArtifactLocation(errPrefix, art)
		if err != nil {
			return nil, err
		}
	}
	return scope, nil
}

func validateWFGlobalParams(tmpl *wfv1.Template, wfGlobalParams []wfv1.Parameter, scope map[string]interface{}) error {
	err := VerifyUniqueNonEmptyNames(wfGlobalParams)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "Workflow spec.arguments.parameters%s", err.Error())
	}
	for _, param := range wfGlobalParams {
		scope[WorkflowGlobalParameterPrefixString+param.Name] = true
	}
	return nil
}

func validateArtifactLocation(errPrefix string, art wfv1.Artifact) error {
	if art.Git != nil {
		if art.Git.Repo == "" {
			return errors.Errorf(errors.CodeBadRequest, "%s.git.repo is required", errPrefix)
		}
	}
	// TODO: validate other artifact locations
	return nil
}

// resolveAllVariables is a helper to ensure all {{variables}} are resolveable from current scope
func resolveAllVariables(scope map[string]interface{}, tmplStr string) error {
	var unresolvedErr error
	_, allowAllItemRefs := scope["item.*"] // 'item.*' is a magic placeholder value set by addItemsToScope
	fstTmpl := fasttemplate.New(tmplStr, "{{", "}}")

	fstTmpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		_, ok := scope[tag]
		if !ok && unresolvedErr == nil {
			if (tag == "item" || strings.HasPrefix(tag, "item.")) && allowAllItemRefs {
				// we are *probably* referencing a undetermined item using withParam
				// NOTE: this is far from foolproof.
			} else {
				unresolvedErr = fmt.Errorf("failed to resolve {{%s}}", tag)
			}
		}
		return 0, nil
	})
	return unresolvedErr
}

func validateLeaf(scope map[string]interface{}, tmpl *wfv1.Template) error {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	err = resolveAllVariables(scope, string(tmplBytes))
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "template '%s' %s", tmpl.Name, err.Error())
	}
	return nil
}

func (ctx *wfValidationCtx) validateSteps(scope map[string]interface{}, tmpl *wfv1.Template) error {
	stepNames := make(map[string]bool)
	for i, stepGroup := range tmpl.Steps {
		for _, step := range stepGroup {
			if step.Name == "" {
				return errors.Errorf(errors.CodeBadRequest, "template '%s' steps[%d].name is required", tmpl.Name, i)
			}
			_, ok := stepNames[step.Name]
			if ok {
				return errors.Errorf(errors.CodeBadRequest, "template '%s' steps[%d].%s name is not unique", tmpl.Name, i, step.Name)
			}
			stepNames[step.Name] = true
			err := addItemsToScope(&step, scope)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "template '%s' steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			stepBytes, err := json.Marshal(stepGroup)
			if err != nil {
				return errors.InternalWrapError(err)
			}
			err = resolveAllVariables(scope, string(stepBytes))
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "template '%s' steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			childTmpl := ctx.wf.GetTemplate(step.Template)
			if childTmpl == nil {
				return errors.Errorf(errors.CodeBadRequest, "template '%s' steps[%d].%s.template '%s' undefined", tmpl.Name, i, step.Name, step.Template)
			}
			err = ctx.validateTemplate(childTmpl, step.Arguments, ctx.wf.Spec.Arguments.Parameters)
			if err != nil {
				return err
			}
		}
		for _, step := range stepGroup {
			ctx.addOutputsToScope(step.Template, step.Name, scope)
		}
	}
	return nil
}

func addItemsToScope(step *wfv1.WorkflowStep, scope map[string]interface{}) error {
	if len(step.WithItems) > 0 && step.WithParam != "" {
		return fmt.Errorf("only one of withItems or withParam can be specified")
	}
	if len(step.WithItems) > 0 {
		switch val := step.WithItems[0].(type) {
		case string, int32, int64, float32, float64:
			scope["item"] = true
		case map[string]interface{}:
			for itemKey := range val {
				scope[fmt.Sprintf("item.%s", itemKey)] = true
			}
		}
	} else if step.WithParam != "" {
		scope["item"] = true
		// 'item.*' is magic placeholder value which resolveAllVariables() will look for
		// when considering if all variables are resolveable.
		scope["item.*"] = true
	}
	return nil
}

func (ctx *wfValidationCtx) addOutputsToScope(templateName string, stepName string, scope map[string]interface{}) {
	tmpl := ctx.wf.GetTemplate(templateName)
	if tmpl.Daemon != nil && *tmpl.Daemon {
		scope[fmt.Sprintf("steps.%s.ip", stepName)] = true
	}
	if tmpl.Script != nil {
		scope[fmt.Sprintf("steps.%s.outputs.result", stepName)] = true
	}
	for _, param := range tmpl.Outputs.Parameters {
		scope[fmt.Sprintf("steps.%s.outputs.parameters.%s", stepName, param.Name)] = true
	}
	for _, art := range tmpl.Outputs.Artifacts {
		scope[fmt.Sprintf("steps.%s.outputs.artifacts.%s", stepName, art.Name)] = true
	}
}

func validateOutputs(tmpl *wfv1.Template) error {
	err := VerifyUniqueNonEmptyNames(tmpl.Outputs.Parameters)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "template '%s' outputs.parameters%s", tmpl.Name, err.Error())
	}
	err = VerifyUniqueNonEmptyNames(tmpl.Outputs.Artifacts)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "template '%s' outputs.artifacts%s", tmpl.Name, err.Error())
	}

	isLeaf := tmpl.Container != nil || tmpl.Script != nil
	for _, art := range tmpl.Outputs.Artifacts {
		artRef := fmt.Sprintf("outputs.artifacts.%s", art.Name)
		if isLeaf {
			if art.Path == "" {
				return errors.Errorf(errors.CodeBadRequest, "template '%s' %s.path not specified", tmpl.Name, artRef)
			}
		} else {
			if art.Path != "" {
				return errors.Errorf(errors.CodeBadRequest, "template '%s' %s.path only valid in container/script templates", tmpl.Name, artRef)
			}
		}
		if art.From != "" {
			return errors.Errorf(errors.CodeBadRequest, "template '%s' %s.from only valid in arguments", tmpl.Name, artRef)
		}
	}
	return nil
}

// VerifyUniqueNonEmptyNames accepts a slice of structs and
// verifies that the Name field of the structs are unique and non-empty
func VerifyUniqueNonEmptyNames(slice interface{}) error {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return errors.InternalErrorf("VerifyNoDuplicateOrEmptyNames given a non-slice type")
	}
	items := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		items[i] = s.Index(i).Interface()
	}
	names := make(map[string]bool)
	getNameFieldValue := func(val interface{}) (string, error) {
		s := reflect.ValueOf(val)
		for i := 0; i < s.NumField(); i++ {
			typeField := s.Type().Field(i)
			if typeField.Name == "Name" {
				return s.Field(i).String(), nil
			}
		}
		return "", errors.InternalError("No 'Name' field in struct")
	}

	for i, item := range items {
		name, err := getNameFieldValue(item)
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("[%d].name is required", i)
		}
		_, ok := names[name]
		if ok {
			return fmt.Errorf("[%d].name '%s' is not unique", i, name)
		}
		names[name] = true
	}
	return nil
}
