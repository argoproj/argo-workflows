package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"

	"github.com/valyala/fasttemplate"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apivalidation "k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/artifacts/hdfs"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/templateresolution"
)

// ValidateOpts provides options when linting
type ValidateOpts struct {
	// Lint indicates if this is performing validation in the context of linting. If true, will
	// skip some validations which is permissible during linting but not submission (e.g. missing
	// input parameters to the workflow)
	Lint bool
	// ContainerRuntimeExecutor will trigger additional validation checks specific to different
	// types of executors. For example, the inability of kubelet/k8s executors to copy artifacts
	// out of the base image layer. If unspecified, will use docker executor validation
	ContainerRuntimeExecutor string
}

// templateValidationCtx is the context for validating a workflow spec
type templateValidationCtx struct {
	ValidateOpts

	// globalParams keeps track of variables which are available the global
	// scope and can be referenced from anywhere.
	globalParams map[string]string
	// results tracks if validation has already been run on a template
	results map[string]bool
	// wf is the Workflow resource which is used to validate templates.
	// It will be omitted in WorkflowTemplate validation.
	wf *wfv1.Workflow
}

func newTemplateValidationCtx(wfClientset wfclientset.Interface, namespace string, wf *wfv1.Workflow, opts ValidateOpts) *templateValidationCtx {
	globalParams := make(map[string]string)
	globalParams[common.GlobalVarWorkflowName] = placeholderValue
	globalParams[common.GlobalVarWorkflowNamespace] = placeholderValue
	globalParams[common.GlobalVarWorkflowUID] = placeholderValue
	return &templateValidationCtx{
		ValidateOpts: opts,
		globalParams: globalParams,
		results:      make(map[string]bool),
		wf:           wf,
	}
}

const (
	// placeholderValue is an arbitrary string to perform mock substitution of variables
	placeholderValue = "placeholder"

	// anyItemMagicValue is a magic value set in addItemsToScope() and checked in
	// resolveAllVariables() to determine if any {{item.name}} can be accepted during
	// variable resolution (to support withParam)
	anyItemMagicValue = "item.*"
)

type FakeArguments struct{}

func (args *FakeArguments) GetParameterByName(name string) *wfv1.Parameter {
	s := placeholderValue
	return &wfv1.Parameter{Name: name, Value: &s}
}

func (args *FakeArguments) GetArtifactByName(name string) *wfv1.Artifact {
	return &wfv1.Artifact{Name: name}
}

var _ wfv1.ArgumentsProvider = &FakeArguments{}

// ValidateWorkflow accepts a workflow and performs validation against it.
func ValidateWorkflow(wfClientset wfclientset.Interface, namespace string, wf *wfv1.Workflow, opts ValidateOpts) error {
	if wf.Namespace != "" {
		namespace = wf.Namespace
	}

	ctx := newTemplateValidationCtx(wfClientset, namespace, wf, opts)
	tmplCtx := templateresolution.NewContextFromClientset(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace), wf)

	err := validateWorkflowFieldNames(wf.Spec.Templates)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "spec.templates%s", err.Error())
	}
	if ctx.Lint {
		// if we are just linting we don't care if spec.arguments.parameters.XXX doesn't have an
		// explicit value. workflows without a default value is a desired use case
		err = validateArgumentsFieldNames("spec.arguments.", wf.Spec.Arguments)
	} else {
		err = validateArguments("spec.arguments.", wf.Spec.Arguments)
	}
	if err != nil {
		return err
	}
	for _, param := range wf.Spec.Arguments.Parameters {
		if param.Name != "" {
			if param.Value != nil {
				ctx.globalParams["workflow.parameters."+param.Name] = *param.Value
			} else {
				ctx.globalParams["workflow.parameters."+param.Name] = placeholderValue
			}
		}
	}

	for k := range wf.ObjectMeta.Annotations {
		ctx.globalParams["workflow.annotations."+k] = placeholderValue
	}
	for k := range wf.ObjectMeta.Labels {
		ctx.globalParams["workflow.labels."+k] = placeholderValue
	}

	if wf.Spec.Entrypoint == "" {
		return errors.New(errors.CodeBadRequest, "spec.entrypoint is required")
	}
	_, err = ctx.validateTemplateHolder(&wfv1.Template{Template: wf.Spec.Entrypoint}, tmplCtx, &wf.Spec.Arguments, map[string]interface{}{})
	if err != nil {
		return err
	}
	if wf.Spec.OnExit != "" {
		// now when validating onExit, {{workflow.status}} is now available as a global
		ctx.globalParams[common.GlobalVarWorkflowStatus] = placeholderValue
		_, err = ctx.validateTemplateHolder(&wfv1.Template{Template: wf.Spec.OnExit}, tmplCtx, &wf.Spec.Arguments, map[string]interface{}{})
		if err != nil {
			return err
		}
	}

	if wf.Spec.PodGC != nil {
		switch wf.Spec.PodGC.Strategy {
		case wfv1.PodGCOnPodCompletion, wfv1.PodGCOnPodSuccess, wfv1.PodGCOnWorkflowCompletion, wfv1.PodGCOnWorkflowSuccess:
		default:
			return errors.Errorf(errors.CodeBadRequest, "podGC.strategy unknown strategy '%s'", wf.Spec.PodGC.Strategy)
		}
	}

	// Check if all templates can be resolved.
	for _, template := range wf.Spec.Templates {
		_, err := ctx.validateTemplateHolder(&wfv1.Template{Template: template.Name}, tmplCtx, &FakeArguments{}, map[string]interface{}{})
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s %s", template.Name, err.Error())
		}
	}
	return nil
}

// ValidateWorkflow accepts a workflow template and performs validation against it.
func ValidateWorkflowTemplate(wfClientset wfclientset.Interface, namespace string, wftmpl *wfv1.WorkflowTemplate) error {
	if wftmpl.Namespace != "" {
		namespace = wftmpl.Namespace
	}
	ctx := newTemplateValidationCtx(wfClientset, namespace, nil, ValidateOpts{})
	tmplCtx := templateresolution.NewContextFromClientset(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace), wftmpl)

	// Check if all templates can be resolved.
	for _, template := range wftmpl.Spec.Templates {
		_, err := ctx.validateTemplateHolder(&wfv1.Template{Template: template.Name}, tmplCtx, &FakeArguments{}, map[string]interface{}{})
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s %s", template.Name, err.Error())
		}
	}
	return nil
}

func (ctx *templateValidationCtx) validateTemplate(tmpl *wfv1.Template, tmplCtx *templateresolution.Context, args wfv1.ArgumentsProvider, extraScope map[string]interface{}) error {
	tmplID := getTemplateID(tmpl)
	_, ok := ctx.results[tmplID]
	if ok {
		// we already processed this template
		return nil
	}
	ctx.results[tmplID] = true

	if hasArguments(tmpl) {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.arguments must be used with template or templateRef", tmpl.Name)
	}

	if err := validateTemplateType(tmpl); err != nil {
		return err
	}

	scope, err := validateInputs(tmpl, extraScope)
	if err != nil {
		return err
	}
	localParams := make(map[string]string)
	if tmpl.IsPodType() {
		localParams[common.LocalVarPodName] = placeholderValue
		scope[common.LocalVarPodName] = placeholderValue
	}
	if tmpl.IsLeaf() {
		for _, art := range tmpl.Outputs.Artifacts {
			if art.Path != "" {
				scope[fmt.Sprintf("outputs.artifacts.%s.path", art.Name)] = true
			}
		}
		for _, param := range tmpl.Outputs.Parameters {
			if param.ValueFrom != nil && param.ValueFrom.Path != "" {
				scope[fmt.Sprintf("outputs.parameters.%s.path", param.Name)] = true
			}
		}
	}

	newTmpl, err := common.ProcessArgs(tmpl, args, ctx.globalParams, localParams, true)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s %s", tmpl.Name, err)
	}
	for globalVar, val := range ctx.globalParams {
		scope[globalVar] = val
	}
	switch newTmpl.GetType() {
	case wfv1.TemplateTypeSteps:
		err = ctx.validateSteps(scope, tmplCtx, newTmpl)
	case wfv1.TemplateTypeDAG:
		err = ctx.validateDAG(scope, tmplCtx, newTmpl)
	default:
		err = ctx.validateLeaf(scope, newTmpl)
	}
	if err != nil {
		return err
	}
	err = validateOutputs(scope, newTmpl)
	if err != nil {
		return err
	}
	err = ctx.validateBaseImageOutputs(newTmpl)
	if err != nil {
		return err
	}
	if newTmpl.ArchiveLocation != nil {
		errPrefix := fmt.Sprintf("templates.%s.archiveLocation", newTmpl.Name)
		err = validateArtifactLocation(errPrefix, *newTmpl.ArchiveLocation)
		if err != nil {
			return err
		}
	}
	return nil
}

// validateTemplateHolder validates a template holder and returns the validated template.
func (ctx *templateValidationCtx) validateTemplateHolder(tmplHolder wfv1.TemplateHolder, tmplCtx *templateresolution.Context, args wfv1.ArgumentsProvider, extraScope map[string]interface{}) (*wfv1.Template, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	tmplName := tmplHolder.GetTemplateName()
	if tmplRef != nil {
		if tmplName != "" {
			return nil, errors.New(errors.CodeBadRequest, "template name cannot be specified with templateRef.")
		}
		if tmplRef.Name == "" {
			return nil, errors.New(errors.CodeBadRequest, "resource name is required")
		}
		if tmplRef.Template == "" {
			return nil, errors.New(errors.CodeBadRequest, "template name is required")
		}
		if tmplRef.RuntimeResolution {
			// Let's see if the template exists at runtime.
			return nil, nil
		}
	} else if tmplName != "" {
		_, err := tmplCtx.GetTemplateByName(tmplName)
		if err != nil {
			if argoerr, ok := err.(errors.ArgoError); ok && argoerr.Code() == errors.CodeNotFound {
				return nil, errors.Errorf(errors.CodeBadRequest, "template name '%s' undefined", tmplName)
			}
			return nil, err
		}
	} else {
		if tmpl, ok := tmplHolder.(*wfv1.Template); ok {
			if tmpl.GetType() != wfv1.TemplateTypeUnknown {
				return nil, errors.New(errors.CodeBadRequest, "template ref can not be used with template type.")
			}
		}
	}

	tmplCtx, resolvedTmpl, err := tmplCtx.ResolveTemplate(tmplHolder, args, ctx.globalParams, map[string]string{}, true)
	if err != nil {
		if argoerr, ok := err.(errors.ArgoError); ok && argoerr.Code() == errors.CodeNotFound {
			if tmplRef != nil {
				return nil, errors.Errorf(errors.CodeBadRequest, "template reference %s.%s not found", tmplRef.Name, tmplRef.Template)
			}
			// this error should not occur.
			return nil, errors.InternalWrapError(err)
		}
		return nil, err
	}

	return resolvedTmpl, ctx.validateTemplate(resolvedTmpl, tmplCtx, args, extraScope)
}

// validateTemplateType validates that only one template type is defined
func validateTemplateType(tmpl *wfv1.Template) error {
	numTypes := 0
	for _, tmplType := range []interface{}{tmpl.TemplateRef, tmpl.Container, tmpl.Steps, tmpl.Script, tmpl.Resource, tmpl.DAG, tmpl.Suspend} {
		if !reflect.ValueOf(tmplType).IsNil() {
			numTypes++
		}
	}
	if tmpl.Template != "" {
		numTypes++
	}
	switch numTypes {
	case 0:
		return errors.Errorf(errors.CodeBadRequest, "templates.%s template type unspecified. choose one of: container, steps, script, resource, dag, suspend, template, template ref", tmpl.Name)
	case 1:
	default:
		return errors.Errorf(errors.CodeBadRequest, "templates.%s multiple template types specified. choose one of: container, steps, script, resource, dag, suspend, template, template ref", tmpl.Name)
	}
	return nil
}

func validateInputs(tmpl *wfv1.Template, extraScope map[string]interface{}) (map[string]interface{}, error) {
	err := validateWorkflowFieldNames(tmpl.Inputs.Parameters)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.inputs.parameters%s", tmpl.Name, err.Error())
	}
	err = validateWorkflowFieldNames(tmpl.Inputs.Artifacts)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.inputs.artifacts%s", tmpl.Name, err.Error())
	}
	scope := make(map[string]interface{})
	for name, value := range extraScope {
		scope[name] = value
	}
	for _, param := range tmpl.Inputs.Parameters {
		scope[fmt.Sprintf("inputs.parameters.%s", param.Name)] = true
	}
	if len(tmpl.Inputs.Parameters) > 0 {
		scope["inputs.parameters"] = true
	}

	for _, art := range tmpl.Inputs.Artifacts {
		artRef := fmt.Sprintf("inputs.artifacts.%s", art.Name)
		scope[artRef] = true
		if tmpl.IsLeaf() {
			if art.Path == "" {
				return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.%s.path not specified", tmpl.Name, artRef)
			}
			scope[fmt.Sprintf("inputs.artifacts.%s.path", art.Name)] = true
		} else {
			if art.Path != "" {
				return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.%s.path only valid in container/script templates", tmpl.Name, artRef)
			}
		}
		if art.From != "" {
			return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.%s.from not valid in inputs", tmpl.Name, artRef)
		}
		errPrefix := fmt.Sprintf("templates.%s.%s", tmpl.Name, artRef)
		err = validateArtifactLocation(errPrefix, art.ArtifactLocation)
		if err != nil {
			return nil, err
		}
	}
	return scope, nil
}

func validateArtifactLocation(errPrefix string, art wfv1.ArtifactLocation) error {
	if art.Git != nil {
		if art.Git.Repo == "" {
			return errors.Errorf(errors.CodeBadRequest, "%s.git.repo is required", errPrefix)
		}
	}
	if art.HDFS != nil {
		err := hdfs.ValidateArtifact(fmt.Sprintf("%s.hdfs", errPrefix), art.HDFS)
		if err != nil {
			return err
		}
	}
	// TODO: validate other artifact locations
	return nil
}

// resolveAllVariables is a helper to ensure all {{variables}} are resolveable from current scope
func resolveAllVariables(scope map[string]interface{}, tmplStr string) error {
	var unresolvedErr error
	_, allowAllItemRefs := scope[anyItemMagicValue] // 'item.*' is a magic placeholder value set by addItemsToScope
	fstTmpl := fasttemplate.New(tmplStr, "{{", "}}")

	fstTmpl.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {

		// Skip the custom variable references
		if !checkValidWorkflowVariablePrefix(tag) {
			return 0, nil
		}
		_, ok := scope[tag]
		if !ok && unresolvedErr == nil {
			if (tag == "item" || strings.HasPrefix(tag, "item.")) && allowAllItemRefs {
				// we are *probably* referencing a undetermined item using withParam
				// NOTE: this is far from foolproof.
			} else if strings.HasPrefix(tag, common.GlobalVarWorkflowCreationTimestamp) {
			} else {
				unresolvedErr = fmt.Errorf("failed to resolve {{%s}}", tag)
			}
		}
		return 0, nil
	})
	return unresolvedErr
}

// checkValidWorkflowVariablePrefix is a helper methood check variable starts workflow root elements
func checkValidWorkflowVariablePrefix(tag string) bool {
	for _, rootTag := range common.GlobalVarValidWorkflowVariablePrefix {
		if strings.HasPrefix(tag, rootTag) {
			return true
		}
	}
	return false
}

func validateNonLeaf(tmpl *wfv1.Template) error {
	if tmpl.ActiveDeadlineSeconds != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.activeDeadlineSeconds is only valid for leaf templates", tmpl.Name)
	}
	if tmpl.RetryStrategy != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.retryStrategy is only valid for container templates", tmpl.Name)
	}
	return nil
}

func (ctx *templateValidationCtx) validateLeaf(scope map[string]interface{}, tmpl *wfv1.Template) error {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	err = resolveAllVariables(scope, string(tmplBytes))
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s: %s", tmpl.Name, err.Error())
	}
	if tmpl.Container != nil {
		// Ensure there are no collisions with volume mountPaths and artifact load paths
		mountPaths := make(map[string]string)
		for i, volMount := range tmpl.Container.VolumeMounts {
			if prev, ok := mountPaths[volMount.MountPath]; ok {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.container.volumeMounts[%d].mountPath '%s' already mounted in %s", tmpl.Name, i, volMount.MountPath, prev)
			}
			mountPaths[volMount.MountPath] = fmt.Sprintf("container.volumeMounts.%s", volMount.Name)
		}
		for i, art := range tmpl.Inputs.Artifacts {
			if prev, ok := mountPaths[art.Path]; ok {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.inputs.artifacts[%d].path '%s' already mounted in %s", tmpl.Name, i, art.Path, prev)
			}
			mountPaths[art.Path] = fmt.Sprintf("inputs.artifacts.%s", art.Name)
		}
	}
	if tmpl.Resource != nil {
		switch tmpl.Resource.Action {
		case "get", "create", "apply", "delete", "replace", "patch":
			// OK
		default:
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.resource.action must be one of: get, create, apply, delete, replace, patch", tmpl.Name)
		}
		// Try to unmarshal the given manifest.
		obj := unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &obj)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.resource.manifest must be a valid yaml", tmpl.Name)
		}
	}
	if tmpl.ActiveDeadlineSeconds != nil {
		if *tmpl.ActiveDeadlineSeconds <= 0 {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.activeDeadlineSeconds must be a positive integer > 0", tmpl.Name)
		}
	}
	if tmpl.Parallelism != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.parallelism is only valid for steps and dag templates", tmpl.Name)
	}
	var automountServiceAccountToken *bool
	if tmpl.AutomountServiceAccountToken != nil {
		automountServiceAccountToken = tmpl.AutomountServiceAccountToken
	} else if ctx.wf != nil && ctx.wf.Spec.AutomountServiceAccountToken != nil {
		automountServiceAccountToken = ctx.wf.Spec.AutomountServiceAccountToken
	}
	executorServiceAccountName := ""
	if tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "" {
		executorServiceAccountName = tmpl.Executor.ServiceAccountName
	} else if ctx.wf != nil && ctx.wf.Spec.Executor != nil && ctx.wf.Spec.Executor.ServiceAccountName != "" {
		executorServiceAccountName = ctx.wf.Spec.Executor.ServiceAccountName
	}
	if automountServiceAccountToken != nil && !*automountServiceAccountToken && executorServiceAccountName == "" {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.executor.serviceAccountName must not be empty if automountServiceAccountToken is false", tmpl.Name)
	}
	return nil
}

func validateArguments(prefix string, arguments wfv1.Arguments) error {
	err := validateArgumentsFieldNames(prefix, arguments)
	if err != nil {
		return err
	}
	return validateArgumentsValues(prefix, arguments)
}

func validateArgumentsFieldNames(prefix string, arguments wfv1.Arguments) error {
	fieldToSlices := map[string]interface{}{
		"parameters": arguments.Parameters,
		"artifacts":  arguments.Artifacts,
	}
	for fieldName, lst := range fieldToSlices {
		err := validateWorkflowFieldNames(lst)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "%s%s%s", prefix, fieldName, err.Error())
		}
	}
	return nil
}

// validateArgumentsValues ensures that all arguments have parameter values or artifact locations
func validateArgumentsValues(prefix string, arguments wfv1.Arguments) error {
	for _, param := range arguments.Parameters {
		if param.Value == nil {
			return errors.Errorf(errors.CodeBadRequest, "%s%s.value is required", prefix, param.Name)
		}
	}
	for _, art := range arguments.Artifacts {
		if art.From == "" && !art.HasLocation() {
			return errors.Errorf(errors.CodeBadRequest, "%s%s.from or artifact location is required", prefix, art.Name)
		}
	}
	return nil
}

func (ctx *templateValidationCtx) validateSteps(scope map[string]interface{}, tmplCtx *templateresolution.Context, tmpl *wfv1.Template) error {
	err := validateNonLeaf(tmpl)
	if err != nil {
		return err
	}
	stepNames := make(map[string]bool)
	resolvedTemplates := make(map[string]*wfv1.Template)
	for i, stepGroup := range tmpl.Steps {
		for _, step := range stepGroup {
			if step.Name == "" {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].name is required", tmpl.Name, i)
			}
			_, ok := stepNames[step.Name]
			if ok {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].name '%s' is not unique", tmpl.Name, i, step.Name)
			}
			if errs := isValidWorkflowFieldName(step.Name); len(errs) != 0 {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].name '%s' is invalid: %s", tmpl.Name, i, step.Name, strings.Join(errs, ";"))
			}
			stepNames[step.Name] = true
			prefix := fmt.Sprintf("steps.%s", step.Name)
			err := addItemsToScope(prefix, step.WithItems, step.WithParam, step.WithSequence, scope)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			stepBytes, err := json.Marshal(stepGroup)
			if err != nil {
				return errors.InternalWrapError(err)
			}
			err = resolveAllVariables(scope, string(stepBytes))
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			err = validateArguments(fmt.Sprintf("templates.%s.steps[%d].%s.arguments.", tmpl.Name, i, step.Name), step.Arguments)
			if err != nil {
				return err
			}
			resolvedTmpl, err := ctx.validateTemplateHolder(&step, tmplCtx, &FakeArguments{}, scope)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			resolvedTemplates[step.Name] = resolvedTmpl
		}
		for i, step := range stepGroup {
			aggregate := len(step.WithItems) > 0 || step.WithParam != ""
			resolvedTmpl := resolvedTemplates[step.Name]
			ctx.addOutputsToScope(resolvedTmpl, fmt.Sprintf("steps.%s", step.Name), scope, aggregate)

			// Validate the template again with actual arguments.
			_, err = ctx.validateTemplateHolder(&step, tmplCtx, &step.Arguments, scope)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
		}
	}
	return nil
}

func addItemsToScope(prefix string, withItems []wfv1.Item, withParam string, withSequence *wfv1.Sequence, scope map[string]interface{}) error {
	defined := 0
	if len(withItems) > 0 {
		defined++
	}
	if withParam != "" {
		defined++
	}
	if withSequence != nil {
		defined++
	}
	if defined > 1 {
		return fmt.Errorf("only one of withItems, withParam, withSequence can be specified")
	}
	if len(withItems) > 0 {
		for i := range withItems {
			switch val := withItems[i].Value.(type) {
			case string, int, int32, int64, float32, float64, bool:
				scope["item"] = true
			case map[string]interface{}:
				for itemKey := range val {
					scope[fmt.Sprintf("item.%s", itemKey)] = true
				}
			default:
				return fmt.Errorf("unsupported withItems type: %v", val)
			}
		}
	} else if withParam != "" {
		scope["item"] = true
		// 'item.*' is magic placeholder value which resolveAllVariables() will look for
		// when considering if all variables are resolveable.
		scope[anyItemMagicValue] = true
	} else if withSequence != nil {
		if withSequence.Count != "" && withSequence.End != "" {
			return errors.New(errors.CodeBadRequest, "only one of count or end can be defined in withSequence")
		}
		scope["item"] = true
	}
	return nil
}

func (ctx *templateValidationCtx) addOutputsToScope(tmpl *wfv1.Template, prefix string, scope map[string]interface{}, aggregate bool) {
	if tmpl.Daemon != nil && *tmpl.Daemon {
		scope[fmt.Sprintf("%s.ip", prefix)] = true
	}
	if tmpl.Script != nil {
		scope[fmt.Sprintf("%s.outputs.result", prefix)] = true
	}
	for _, param := range tmpl.Outputs.Parameters {
		scope[fmt.Sprintf("%s.outputs.parameters.%s", prefix, param.Name)] = true
		if param.GlobalName != "" && !isParameter(param.GlobalName) {
			globalParamName := fmt.Sprintf("workflow.outputs.parameters.%s", param.GlobalName)
			scope[globalParamName] = true
			ctx.globalParams[globalParamName] = placeholderValue
		}
	}
	for _, art := range tmpl.Outputs.Artifacts {
		scope[fmt.Sprintf("%s.outputs.artifacts.%s", prefix, art.Name)] = true
		if art.GlobalName != "" && !isParameter(art.GlobalName) {
			globalArtName := fmt.Sprintf("workflow.outputs.artifacts.%s", art.GlobalName)
			scope[globalArtName] = true
			ctx.globalParams[globalArtName] = placeholderValue
		}
	}
	if aggregate {
		switch tmpl.GetType() {
		case wfv1.TemplateTypeScript:
			scope[fmt.Sprintf("%s.outputs.result", prefix)] = true
		default:
			scope[fmt.Sprintf("%s.outputs.parameters", prefix)] = true
		}
	}
}

func validateOutputs(scope map[string]interface{}, tmpl *wfv1.Template) error {
	err := validateWorkflowFieldNames(tmpl.Outputs.Parameters)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs.parameters %s", tmpl.Name, err.Error())
	}
	err = validateWorkflowFieldNames(tmpl.Outputs.Artifacts)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs.artifacts %s", tmpl.Name, err.Error())
	}
	outputBytes, err := json.Marshal(tmpl.Outputs)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	err = resolveAllVariables(scope, string(outputBytes))
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs %s", tmpl.Name, err.Error())
	}

	for _, art := range tmpl.Outputs.Artifacts {
		artRef := fmt.Sprintf("outputs.artifacts.%s", art.Name)
		if tmpl.IsLeaf() {
			if art.Path == "" {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.%s.path not specified", tmpl.Name, artRef)
			}
		} else {
			if art.Path != "" {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.%s.path only valid in container/script templates", tmpl.Name, artRef)
			}
		}
		if art.GlobalName != "" && !isParameter(art.GlobalName) {
			errs := isValidParamOrArtifactName(art.GlobalName)
			if len(errs) > 0 {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.%s.globalName: %s", tmpl.Name, artRef, errs[0])
			}
		}
	}
	for _, param := range tmpl.Outputs.Parameters {
		paramRef := fmt.Sprintf("templates.%s.outputs.parameters.%s", tmpl.Name, param.Name)
		err = validateOutputParameter(paramRef, &param)
		if err != nil {
			return err
		}
		if param.ValueFrom != nil {
			tmplType := tmpl.GetType()
			switch tmplType {
			case wfv1.TemplateTypeContainer, wfv1.TemplateTypeScript:
				if param.ValueFrom.Path == "" {
					return errors.Errorf(errors.CodeBadRequest, "%s.path must be specified for %s templates", paramRef, tmplType)
				}
			case wfv1.TemplateTypeResource:
				if param.ValueFrom.JQFilter == "" && param.ValueFrom.JSONPath == "" {
					return errors.Errorf(errors.CodeBadRequest, "%s .jqFilter or jsonPath must be specified for %s templates", paramRef, tmplType)
				}
			case wfv1.TemplateTypeDAG, wfv1.TemplateTypeSteps:
				if param.ValueFrom.Parameter == "" {
					return errors.Errorf(errors.CodeBadRequest, "%s.parameter must be specified for %s templates", paramRef, tmplType)
				}
			}
		}
		if param.GlobalName != "" && !isParameter(param.GlobalName) {
			errs := isValidParamOrArtifactName(param.GlobalName)
			if len(errs) > 0 {
				return errors.Errorf(errors.CodeBadRequest, "%s.globalName: %s", paramRef, errs[0])
			}
		}
	}
	return nil
}

// validateBaseImageOutputs detects if the template contains an valid output from base image layer
func (ctx *templateValidationCtx) validateBaseImageOutputs(tmpl *wfv1.Template) error {
	switch ctx.ContainerRuntimeExecutor {
	case "", common.ContainerRuntimeExecutorDocker:
		// docker executor supports all modes of artifact outputs
	case common.ContainerRuntimeExecutorPNS:
		// pns supports copying from the base image, but only if there is no volume mount underneath it
		errMsg := "pns executor does not support outputs from base image layer with volume mounts. must use emptyDir"
		for _, out := range tmpl.Outputs.Artifacts {
			if common.FindOverlappingVolume(tmpl, out.Path) == nil {
				// output is in the base image layer. need to verify there are no volume mounts under it
				if tmpl.Container != nil {
					for _, volMnt := range tmpl.Container.VolumeMounts {
						if strings.HasPrefix(volMnt.MountPath, out.Path+"/") {
							return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs.artifacts.%s: %s", tmpl.Name, out.Name, errMsg)
						}
					}

				}
				if tmpl.Script != nil {
					for _, volMnt := range tmpl.Script.VolumeMounts {
						if strings.HasPrefix(volMnt.MountPath, out.Path+"/") {
							return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs.artifacts.%s: %s", tmpl.Name, out.Name, errMsg)
						}
					}
				}
			}
		}
	case common.ContainerRuntimeExecutorK8sAPI, common.ContainerRuntimeExecutorKubelet:
		// for kubelet/k8s fail validation if we detect artifact is copied from base image layer
		errMsg := fmt.Sprintf("%s executor does not support outputs from base image layer. must use emptyDir", ctx.ContainerRuntimeExecutor)
		for _, out := range tmpl.Outputs.Artifacts {
			if common.FindOverlappingVolume(tmpl, out.Path) == nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs.artifacts.%s: %s", tmpl.Name, out.Name, errMsg)
			}
		}
		for _, out := range tmpl.Outputs.Parameters {
			if out.ValueFrom != nil && common.FindOverlappingVolume(tmpl, out.ValueFrom.Path) == nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs.parameters.%s: %s", tmpl.Name, out.Name, errMsg)
			}
		}
	}
	return nil
}

// validateOutputParameter verifies that only one of valueFrom is defined in an output
func validateOutputParameter(paramRef string, param *wfv1.Parameter) error {
	if param.ValueFrom != nil && param.Value != nil {
		return errors.Errorf(errors.CodeBadRequest, "%s has both valueFrom and value specified. Choose one.", paramRef)
	}
	if param.Value != nil {
		return nil
	}
	if param.ValueFrom == nil {
		return errors.Errorf(errors.CodeBadRequest, "%s does not have valueFrom or value specified", paramRef)
	}
	paramTypes := 0
	for _, value := range []string{param.ValueFrom.Path, param.ValueFrom.JQFilter, param.ValueFrom.JSONPath, param.ValueFrom.Parameter} {
		if value != "" {
			paramTypes++
		}
	}
	switch paramTypes {
	case 0:
		return errors.New(errors.CodeBadRequest, "valueFrom type unspecified. choose one of: path, jqFilter, jsonPath, parameter")
	case 1:
	default:
		return errors.New(errors.CodeBadRequest, "multiple valueFrom types specified. choose one of: path, jqFilter, jsonPath, parameter")
	}
	return nil
}

// validateWorkflowFieldNames accepts a slice of structs and
// verifies that the Name field of the structs are:
// * unique
// * non-empty
// * matches matches our regex requirements
func validateWorkflowFieldNames(slice interface{}) error {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return errors.InternalErrorf("validateWorkflowFieldNames given a non-slice type")
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
			return errors.Errorf(errors.CodeBadRequest, "[%d].name is required", i)
		}
		var errs []string
		t := reflect.TypeOf(item)
		if t == reflect.TypeOf(wfv1.Parameter{}) || t == reflect.TypeOf(wfv1.Artifact{}) {
			errs = isValidParamOrArtifactName(name)
		} else {
			errs = isValidWorkflowFieldName(name)
		}
		if len(errs) != 0 {
			return errors.Errorf(errors.CodeBadRequest, "[%d].name: '%s' is invalid: %s", i, name, strings.Join(errs, ";"))
		}
		_, ok := names[name]
		if ok {
			return errors.Errorf(errors.CodeBadRequest, "[%d].name '%s' is not unique", i, name)
		}
		names[name] = true
	}
	return nil
}

func (ctx *templateValidationCtx) validateDAG(scope map[string]interface{}, tmplCtx *templateresolution.Context, tmpl *wfv1.Template) error {
	err := validateNonLeaf(tmpl)
	if err != nil {
		return err
	}
	err = validateWorkflowFieldNames(tmpl.DAG.Tasks)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks%s", tmpl.Name, err.Error())
	}
	nameToTask := make(map[string]wfv1.DAGTask)
	for _, task := range tmpl.DAG.Tasks {
		nameToTask[task.Name] = task
	}
	resolvedTemplates := make(map[string]*wfv1.Template)

	// Verify dependencies for all tasks can be resolved as well as template names
	for _, task := range tmpl.DAG.Tasks {
		resolvedTmpl, err := ctx.validateTemplateHolder(&task, tmplCtx, &FakeArguments{}, map[string]interface{}{})
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
		prefix := fmt.Sprintf("tasks.%s", task.Name)
		ctx.addOutputsToScope(resolvedTmpl, prefix, scope, false)
		resolvedTemplates[task.Name] = resolvedTmpl
		dupDependencies := make(map[string]bool)
		for j, depName := range task.Dependencies {
			if _, ok := dupDependencies[depName]; ok {
				return errors.Errorf(errors.CodeBadRequest,
					"templates.%s.tasks.%s.dependencies[%d] dependency '%s' duplicated",
					tmpl.Name, task.Name, j, depName)
			}
			dupDependencies[depName] = true
			if _, ok := nameToTask[depName]; !ok {
				return errors.Errorf(errors.CodeBadRequest,
					"templates.%s.tasks.%s.dependencies[%d] dependency '%s' not defined",
					tmpl.Name, task.Name, j, depName)
			}
		}
	}

	if err = verifyNoCycles(tmpl, nameToTask); err != nil {
		return err
	}

	err = resolveAllVariables(scope, tmpl.DAG.Target)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.targets %s", tmpl.Name, err.Error())
	}
	if err = validateDAGTargets(tmpl, nameToTask); err != nil {
		return err
	}

	for _, task := range tmpl.DAG.Tasks {
		resolvedTmpl := resolvedTemplates[task.Name]
		// add all tasks outputs to scope so that a nested DAGs can have outputs
		prefix := fmt.Sprintf("tasks.%s", task.Name)
		ctx.addOutputsToScope(resolvedTmpl, prefix, scope, false)
		taskBytes, err := json.Marshal(task)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		taskScope := make(map[string]interface{})
		for k, v := range scope {
			taskScope[k] = v
		}
		ancestry := common.GetTaskAncestry(nil, task.Name, tmpl.DAG.Tasks)
		for _, ancestor := range ancestry {
			ancestorTask := nameToTask[ancestor]
			resolvedTmpl := resolvedTemplates[ancestor]
			ancestorPrefix := fmt.Sprintf("tasks.%s", ancestor)
			aggregate := len(ancestorTask.WithItems) > 0 || ancestorTask.WithParam != ""
			ctx.addOutputsToScope(resolvedTmpl, ancestorPrefix, taskScope, aggregate)
		}
		err = addItemsToScope(prefix, task.WithItems, task.WithParam, task.WithSequence, taskScope)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
		err = resolveAllVariables(taskScope, string(taskBytes))
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
		err = validateArguments(fmt.Sprintf("templates.%s.tasks.%s.arguments.", tmpl.Name, task.Name), task.Arguments)
		if err != nil {
			return err
		}
		// Validate the template again with actual arguments.
		_, err = ctx.validateTemplateHolder(&task, tmplCtx, &task.Arguments, scope)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
	}

	return nil
}

func validateDAGTargets(tmpl *wfv1.Template, nameToTask map[string]wfv1.DAGTask) error {
	if tmpl.DAG.Target == "" {
		return nil
	}
	for _, targetName := range strings.Split(tmpl.DAG.Target, " ") {
		if isParameter(targetName) {
			continue
		}
		if _, ok := nameToTask[targetName]; !ok {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.targets: target '%s' is not defined", tmpl.Name, targetName)
		}
	}
	return nil
}

// verifyNoCycles verifies there are no cycles in the DAG graph
func verifyNoCycles(tmpl *wfv1.Template, nameToTask map[string]wfv1.DAGTask) error {
	visited := make(map[string]bool)
	var noCyclesHelper func(taskName string, cycle []string) error
	noCyclesHelper = func(taskName string, cycle []string) error {
		if _, ok := visited[taskName]; ok {
			return nil
		}
		task := nameToTask[taskName]
		for _, depName := range task.Dependencies {
			for _, name := range cycle {
				if name == depName {
					return errors.Errorf(errors.CodeBadRequest,
						"templates.%s.tasks dependency cycle detected: %s->%s",
						tmpl.Name, strings.Join(cycle, "->"), name)
				}
			}
			cycle = append(cycle, depName)
			err := noCyclesHelper(depName, cycle)
			if err != nil {
				return err
			}
			cycle = cycle[0 : len(cycle)-1]
		}
		visited[taskName] = true
		return nil
	}

	for _, task := range tmpl.DAG.Tasks {
		err := noCyclesHelper(task.Name, []string{})
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	// paramRegex matches a parameter. e.g. {{inputs.parameters.blah}}
	paramRegex               = regexp.MustCompile(`{{[-a-zA-Z0-9]+(\.[-a-zA-Z0-9_]+)*}}`)
	paramOrArtifactNameRegex = regexp.MustCompile(`^[-a-zA-Z0-9_]+[-a-zA-Z0-9_]*$`)
)

func isParameter(p string) bool {
	return paramRegex.MatchString(p)
}

func isValidParamOrArtifactName(p string) []string {
	var errs []string
	if !paramOrArtifactNameRegex.MatchString(p) {
		return append(errs, "Parameter/Artifact name must consist of alpha-numeric characters, '_' or '-' e.g. my_param_1, MY-PARAM-1")
	}
	return errs
}

func hasArguments(tmpl *wfv1.Template) bool {
	return len(tmpl.Arguments.Parameters) > 0 || len(tmpl.Arguments.Artifacts) > 0
}

const (
	workflowFieldNameFmt    string = "[a-zA-Z0-9][-a-zA-Z0-9]*"
	workflowFieldNameErrMsg string = "name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character"
	workflowFieldMaxLength  int    = 128
)

var workflowFieldNameRegexp = regexp.MustCompile("^" + workflowFieldNameFmt + "$")

// isValidWorkflowFieldName : workflow field name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character
func isValidWorkflowFieldName(name string) []string {
	var errs []string
	if len(name) > workflowFieldMaxLength {
		errs = append(errs, apivalidation.MaxLenError(workflowFieldMaxLength))
	}
	if !workflowFieldNameRegexp.MatchString(name) {
		msg := workflowFieldNameErrMsg + " (e.g. My-name1-2, 123-NAME)"
		errs = append(errs, msg)
	}
	return errs
}

func getTemplateID(tmpl *wfv1.Template) string {
	return fmt.Sprintf("%s %v", tmpl.Name, tmpl.TemplateRef)
}
