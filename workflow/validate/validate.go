package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apivalidation "k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util"
	"github.com/argoproj/argo/util/help"
	"github.com/argoproj/argo/util/intstr"
	"github.com/argoproj/argo/util/sorting"
	"github.com/argoproj/argo/workflow/artifacts/hdfs"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/metrics"
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

	// IgnoreEntrypoint indicates to skip/ignore the EntryPoint validation on workflow spec.
	// Entrypoint is optional for WorkflowTemplate and ClusterWorkflowTemplate
	IgnoreEntrypoint bool

	// WorkflowTemplateValidation indicates that the current context is validating a WorkflowTemplate or ClusterWorkflowTemplate
	WorkflowTemplateValidation bool
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

func newTemplateValidationCtx(wf *wfv1.Workflow, opts ValidateOpts) *templateValidationCtx {
	globalParams := make(map[string]string)
	globalParams[common.GlobalVarWorkflowName] = placeholderGenerator.NextPlaceholder()
	globalParams[common.GlobalVarWorkflowNamespace] = placeholderGenerator.NextPlaceholder()
	globalParams[common.GlobalVarWorkflowServiceAccountName] = placeholderGenerator.NextPlaceholder()
	globalParams[common.GlobalVarWorkflowUID] = placeholderGenerator.NextPlaceholder()
	return &templateValidationCtx{
		ValidateOpts: opts,
		globalParams: globalParams,
		results:      make(map[string]bool),
		wf:           wf,
	}
}

const (
	// anyItemMagicValue is a magic value set in addItemsToScope() and checked in
	// resolveAllVariables() to determine if any {{item.name}} can be accepted during
	// variable resolution (to support withParam)
	anyItemMagicValue                    = "item.*"
	anyWorkflowOutputParameterMagicValue = "workflow.outputs.parameters.*"
	anyWorkflowOutputArtifactMagicValue  = "workflow.outputs.artifacts.*"
)

var (
	placeholderGenerator = common.NewPlaceholderGenerator()
)

type FakeArguments struct{}

func (args *FakeArguments) GetParameterByName(name string) *wfv1.Parameter {
	s := placeholderGenerator.NextPlaceholder()
	return &wfv1.Parameter{Name: name, Value: wfv1.AnyStringPtr(s)}
}

func (args *FakeArguments) GetArtifactByName(name string) *wfv1.Artifact {
	return &wfv1.Artifact{Name: name}
}

var _ wfv1.ArgumentsProvider = &FakeArguments{}

// ValidateWorkflow accepts a workflow and performs validation against it.
func ValidateWorkflow(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, wf *wfv1.Workflow, opts ValidateOpts) (*wfv1.Conditions, error) {
	wfConditions := &wfv1.Conditions{}
	ctx := newTemplateValidationCtx(wf, opts)
	tmplCtx := templateresolution.NewContext(wftmplGetter, cwftmplGetter, wf, wf)

	var wfSpecHolder wfv1.WorkflowSpecHolder
	var wfTmplRef *wfv1.TemplateRef
	var err error

	entrypoint := wf.Spec.Entrypoint

	hasWorkflowTemplateRef := wf.Spec.WorkflowTemplateRef != nil

	if hasWorkflowTemplateRef {
		err := ValidateWorkflowTemplateRefFields(wf.Spec)
		if err != nil {
			return nil, err
		}
		if wf.Spec.WorkflowTemplateRef.ClusterScope {
			wfSpecHolder, err = cwftmplGetter.Get(wf.Spec.WorkflowTemplateRef.Name)
		} else {
			wfSpecHolder, err = wftmplGetter.Get(wf.Spec.WorkflowTemplateRef.Name)
		}
		if err != nil {
			return nil, err
		}
		if entrypoint == "" {
			entrypoint = wfSpecHolder.GetWorkflowSpec().Entrypoint
		}
		wfTmplRef = wf.Spec.WorkflowTemplateRef.ToTemplateRef(entrypoint)
	}
	err = validateWorkflowFieldNames(wf.Spec.Templates)

	wfArgs := wf.Spec.Arguments

	if wf.Spec.WorkflowTemplateRef != nil {
		wfArgs.Parameters = util.MergeParameters(wfArgs.Parameters, wfSpecHolder.GetWorkflowSpec().Arguments.Parameters)
		wfArgs.Artifacts = util.MergeArtifacts(wfArgs.Artifacts, wfSpecHolder.GetWorkflowSpec().Arguments.Artifacts)
	}
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "spec.templates%s", err.Error())
	}
	if ctx.Lint {
		// if we are just linting we don't care if spec.arguments.parameters.XXX doesn't have an
		// explicit value. workflows without a default value is a desired use case
		err = validateArgumentsFieldNames("spec.arguments.", wfArgs)
	} else {
		err = validateArguments("spec.arguments.", wfArgs)
	}
	if err != nil {
		return nil, err
	}
	if len(wfArgs.Parameters) > 0 {
		ctx.globalParams[common.GlobalVarWorkflowParameters] = placeholderGenerator.NextPlaceholder()
	}

	for _, param := range wfArgs.Parameters {
		if param.Name != "" {
			if param.Value != nil {
				ctx.globalParams["workflow.parameters."+param.Name] = param.Value.String()
			} else {
				ctx.globalParams["workflow.parameters."+param.Name] = placeholderGenerator.NextPlaceholder()
			}
		}
	}

	for k := range wf.ObjectMeta.Annotations {
		ctx.globalParams["workflow.annotations."+k] = placeholderGenerator.NextPlaceholder()
	}
	for k := range wf.ObjectMeta.Labels {
		ctx.globalParams["workflow.labels."+k] = placeholderGenerator.NextPlaceholder()
	}

	if wf.Spec.Priority != nil {
		ctx.globalParams[common.GlobalVarWorkflowPriority] = strconv.Itoa(int(*wf.Spec.Priority))
	}

	if !opts.IgnoreEntrypoint && entrypoint == "" {
		return nil, errors.New(errors.CodeBadRequest, "spec.entrypoint is required")
	}

	// Make sure that templates are not defined with deprecated fields
	for _, template := range wf.Spec.Templates {
		if !template.Arguments.IsEmpty() {
			logrus.Warn("template.arguments is deprecated and its contents are ignored")
			wfConditions.UpsertConditionMessage(wfv1.Condition{
				Type:    wfv1.ConditionTypeSpecWarning,
				Status:  v1.ConditionTrue,
				Message: fmt.Sprintf(`template.arguments is deprecated and its contents are ignored. See more: %s`, help.WorkflowTemplatesReferencingOtherTemplates),
			})
		}
		if template.TemplateRef != nil {
			logrus.Warn(getTemplateRefHelpString(&template))
			wfConditions.UpsertConditionMessage(wfv1.Condition{
				Type:    wfv1.ConditionTypeSpecWarning,
				Status:  v1.ConditionTrue,
				Message: fmt.Sprintf(`Referencing/calling other templates directly on a "template" is deprecated; they should be referenced in a "steps" or a "dag" template. See more: %s`, help.WorkflowTemplatesReferencingOtherTemplates),
			})
		}
		if template.Template != "" {
			logrus.Warn(getTemplateRefHelpString(&template))
			wfConditions.UpsertConditionMessage(wfv1.Condition{
				Type:    wfv1.ConditionTypeSpecWarning,
				Status:  v1.ConditionTrue,
				Message: fmt.Sprintf(`Referencing/calling other templates directly on a "template" is deprecated; they should be referenced in a "steps" or a "dag" template. See more: %s`, help.WorkflowTemplatesReferencingOtherTemplates),
			})
		}
	}

	if !opts.IgnoreEntrypoint {
		var args wfv1.ArgumentsProvider
		args = &wfArgs
		if opts.WorkflowTemplateValidation {
			args = &FakeArguments{}
		}
		tmpl := &wfv1.WorkflowStep{Template: entrypoint}
		if hasWorkflowTemplateRef {
			tmpl = &wfv1.WorkflowStep{TemplateRef: wfTmplRef}
		}
		_, err = ctx.validateTemplateHolder(tmpl, tmplCtx, args)
		if err != nil {
			return nil, err
		}
	}
	if wf.Spec.OnExit != "" {
		// now when validating onExit, {{workflow.status}} is now available as a global
		ctx.globalParams[common.GlobalVarWorkflowStatus] = placeholderGenerator.NextPlaceholder()
		ctx.globalParams[common.GlobalVarWorkflowFailures] = placeholderGenerator.NextPlaceholder()
		_, err = ctx.validateTemplateHolder(&wfv1.WorkflowStep{Template: wf.Spec.OnExit}, tmplCtx, &wf.Spec.Arguments)
		if err != nil {
			return nil, err
		}
	}

	if wf.Spec.PodGC != nil {
		switch wf.Spec.PodGC.Strategy {
		case wfv1.PodGCOnPodCompletion, wfv1.PodGCOnPodSuccess, wfv1.PodGCOnWorkflowCompletion, wfv1.PodGCOnWorkflowSuccess:
		default:
			return nil, errors.Errorf(errors.CodeBadRequest, "podGC.strategy unknown strategy '%s'", wf.Spec.PodGC.Strategy)
		}
	}

	// Check if all templates can be resolved.
	for _, template := range wf.Spec.Templates {
		_, err := ctx.validateTemplateHolder(&wfv1.WorkflowStep{Template: template.Name}, tmplCtx, &FakeArguments{})
		if err != nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s %s", template.Name, err.Error())
		}
	}
	return wfConditions, nil
}

func ValidateWorkflowTemplateRefFields(wfSpec wfv1.WorkflowSpec) error {
	if len(wfSpec.Templates) > 0 {
		return errors.Errorf(errors.CodeBadRequest, "Templates is invalid field in spec if workflow referred WorkflowTemplate reference")
	}
	return nil
}

// ValidateWorkflowTemplate accepts a workflow template and performs validation against it.
func ValidateWorkflowTemplate(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, wftmpl *wfv1.WorkflowTemplate) (*wfv1.Conditions, error) {
	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Labels:      wftmpl.ObjectMeta.Labels,
			Annotations: wftmpl.ObjectMeta.Annotations,
		},
		Spec: wftmpl.Spec.WorkflowSpec,
	}
	return ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{IgnoreEntrypoint: wf.Spec.Entrypoint == "", WorkflowTemplateValidation: true})
}

// ValidateClusterWorkflowTemplate accepts a cluster workflow template and performs validation against it.
func ValidateClusterWorkflowTemplate(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, cwftmpl *wfv1.ClusterWorkflowTemplate) (*wfv1.Conditions, error) {
	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Labels:      cwftmpl.ObjectMeta.Labels,
			Annotations: cwftmpl.ObjectMeta.Annotations,
		},
		Spec: cwftmpl.Spec.WorkflowSpec,
	}
	return ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{IgnoreEntrypoint: wf.Spec.Entrypoint == "", WorkflowTemplateValidation: true})
}

// ValidateCronWorkflow validates a CronWorkflow
func ValidateCronWorkflow(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, cronWf *wfv1.CronWorkflow) error {
	if _, err := cron.ParseStandard(cronWf.Spec.Schedule); err != nil {
		return errors.Errorf(errors.CodeBadRequest, "cron schedule is malformed: %s", err)
	}

	switch cronWf.Spec.ConcurrencyPolicy {
	case wfv1.AllowConcurrent, wfv1.ForbidConcurrent, wfv1.ReplaceConcurrent, "":
		// Do nothing
	default:
		return errors.Errorf(errors.CodeBadRequest, "'%s' is not a valid concurrencyPolicy", cronWf.Spec.ConcurrencyPolicy)
	}

	if cronWf.Spec.StartingDeadlineSeconds != nil && *cronWf.Spec.StartingDeadlineSeconds < 0 {
		return errors.Errorf(errors.CodeBadRequest, "startingDeadlineSeconds must be positive")
	}

	wf := common.ConvertCronWorkflowToWorkflow(cronWf)

	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "cannot validate Workflow: %s", err)
	}
	return nil
}

func getTemplateRefHelpString(tmpl *wfv1.Template) string {
	out := `Referencing/calling other templates directly on a "template" is deprecated, no longer supported, and will be removed in a future version.

Templates should be referenced from within a "steps" or a "dag" template. Here is how you would reference this on a "steps" template:

- name: %s
  steps:
    - - name: call-%s`

	if tmpl.TemplateRef != nil {
		out += `
        templateRef:
          name: %s
          template: %s`

		out = fmt.Sprintf(out, tmpl.Name, tmpl.TemplateRef.Template, tmpl.TemplateRef.Name, tmpl.TemplateRef.Template)
	} else if tmpl.Template != "" {
		out += `
        template: %s`

		out = fmt.Sprintf(out, tmpl.Name, tmpl.Template, tmpl.Template)
	}

	if !tmpl.Inputs.IsEmpty() {
		out += `
        arguments:    # Inputs should be converted to arguments`
		inputBytes, err := yaml.Marshal(tmpl.Inputs)
		if err != nil {
			panic(err)
		}
		for _, line := range strings.Split(string(inputBytes), "\n") {
			out += `
          ` + line
		}
	}

	out += `

For more information, see: %s

`

	out = fmt.Sprintf(out, help.WorkflowTemplatesReferencingOtherTemplates)
	return out
}

func (ctx *templateValidationCtx) validateTemplate(tmpl *wfv1.Template, tmplCtx *templateresolution.Context, args wfv1.ArgumentsProvider) error {
	if err := validateTemplateType(tmpl); err != nil {
		return err
	}

	scope, err := validateInputs(tmpl)
	if err != nil {
		return err
	}

	localParams := make(map[string]string)
	if tmpl.IsPodType() {
		localParams[common.LocalVarPodName] = placeholderGenerator.NextPlaceholder()
		scope[common.LocalVarPodName] = placeholderGenerator.NextPlaceholder()
	}
	if tmpl.RetryStrategy != nil {
		localParams[common.LocalVarRetries] = placeholderGenerator.NextPlaceholder()
		scope[common.LocalVarRetries] = placeholderGenerator.NextPlaceholder()
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

	if newTmpl.Timeout != "" {
		if !newTmpl.IsLeaf() {
			return fmt.Errorf("%s template doesn't support timeout field.", newTmpl.GetType())
		}
		// Check timeout should not be a whole number
		_, err := strconv.Atoi(newTmpl.Timeout)
		if err == nil {
			return fmt.Errorf("%s has invalid duration format in timeout.", newTmpl.Name)
		}

	}

	tmplID := getTemplateID(tmpl)
	_, ok := ctx.results[tmplID]
	if ok {
		// we can skip the rest since it has been validated.
		return nil
	}
	ctx.results[tmplID] = true

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
	if newTmpl.Metrics != nil {
		for _, metric := range newTmpl.Metrics.Prometheus {
			if !metrics.IsValidMetricName(metric.Name) {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s metric name '%s' is invalid. Metric names must contain alphanumeric characters, '_', or ':'", tmpl.Name, metric.Name)
			}
			if err := metrics.ValidateMetricLabels(metric.GetMetricLabels()); err != nil {
				return err
			}
			if metric.Help == "" {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s metric '%s' must contain a help string under 'help: ' field", tmpl.Name, metric.Name)
			}
		}
	}
	return nil
}

// validateTemplateHolder validates a template holder and returns the validated template.
func (ctx *templateValidationCtx) validateTemplateHolder(tmplHolder wfv1.TemplateReferenceHolder, tmplCtx *templateresolution.Context, args wfv1.ArgumentsProvider) (*wfv1.Template, error) {
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
			logrus.Warnf("the 'runtimeResolution' field is deprecated and ignored")
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

	tmplCtx, resolvedTmpl, _, err := tmplCtx.ResolveTemplate(tmplHolder)
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

	// Validate retryStrategy
	if resolvedTmpl.RetryStrategy != nil {
		switch resolvedTmpl.RetryStrategy.RetryPolicy {
		case wfv1.RetryPolicyAlways, wfv1.RetryPolicyOnError, wfv1.RetryPolicyOnFailure, "":
			// Passes validation
		default:
			return nil, fmt.Errorf("%s is not a valid RetryPolicy", resolvedTmpl.RetryStrategy.RetryPolicy)
		}
	}

	return resolvedTmpl, ctx.validateTemplate(resolvedTmpl, tmplCtx, args)
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
		// Do nothing
	default:
		return errors.Errorf(errors.CodeBadRequest, "templates.%s multiple template types specified. choose one of: container, steps, script, resource, dag, suspend, template, template ref", tmpl.Name)
	}
	return nil
}

func validateInputs(tmpl *wfv1.Template) (map[string]interface{}, error) {
	err := validateWorkflowFieldNames(tmpl.Inputs.Parameters)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.inputs.parameters%s", tmpl.Name, err.Error())
	}
	err = validateWorkflowFieldNames(tmpl.Inputs.Artifacts)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.inputs.artifacts%s", tmpl.Name, err.Error())
	}
	scope := make(map[string]interface{})
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
	_, allowAllWorkflowOutputParameterRefs := scope[anyWorkflowOutputParameterMagicValue]
	_, allowAllWorkflowOutputArtifactRefs := scope[anyWorkflowOutputArtifactMagicValue]
	fstTmpl, err := fasttemplate.NewTemplate(tmplStr, "{{", "}}")
	if err != nil {
		return fmt.Errorf("unable to parse argo varaible: %w", err)
	}

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
			} else if strings.HasPrefix(tag, "workflow.outputs.parameters.") && allowAllWorkflowOutputParameterRefs {
				// Allow runtime resolution of workflow output parameter names
			} else if strings.HasPrefix(tag, "workflow.outputs.artifacts.") && allowAllWorkflowOutputArtifactRefs {
				// Allow runtime resolution of workflow output artifact names
			} else if strings.HasPrefix(tag, "outputs.") {
				// We are self referencing for metric emission, allow it.
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
		if tmpl.Container.Image == "" {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.container.image may not be empty", tmpl.Name)
		}
	}
	if tmpl.Resource != nil {
		if !placeholderGenerator.IsPlaceholder(tmpl.Resource.Action) {
			switch tmpl.Resource.Action {
			case "get", "create", "apply", "delete", "replace", "patch":
				// OK
			default:
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.resource.action must be one of: get, create, apply, delete, replace, patch", tmpl.Name)
			}
		}
		if !placeholderGenerator.IsPlaceholder(tmpl.Resource.Manifest) {
			// Try to unmarshal the given manifest.
			obj := unstructured.Unstructured{}
			err := yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &obj)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.resource.manifest must be a valid yaml", tmpl.Name)
			}
		}
	}
	if tmpl.Script != nil {
		if tmpl.Script.Image == "" {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.script.image may not be empty", tmpl.Name)
		}
	}
	if tmpl.ActiveDeadlineSeconds != nil {
		if !intstr.IsValidIntOrArgoVariable(tmpl.ActiveDeadlineSeconds) && !placeholderGenerator.IsPlaceholder(tmpl.ActiveDeadlineSeconds.StrVal) {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.activeDeadlineSeconds must be a positive integer > 0 or an argo variable", tmpl.Name)
		}
		if i, err := intstr.Int(tmpl.ActiveDeadlineSeconds); err == nil && i != nil && *i < 0 {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.activeDeadlineSeconds must be a positive integer > 0 or an argo variable", tmpl.Name)
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
		if param.Enum != nil {
			if len(param.Enum) == 0 {
				return errors.Errorf(errors.CodeBadRequest, "%s%s.enum should contain at least one value", prefix, param.Name)
			}
			valueSpecifiedInEnumList := false
			for _, enum := range param.Enum {
				if enum == *param.Value {
					valueSpecifiedInEnumList = true
					break
				}
			}
			if !valueSpecifiedInEnumList {
				return errors.Errorf(errors.CodeBadRequest, "%s%s.value should be present in %s%s.enum list", prefix, param.Name, prefix, param.Name)
			}
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
		for _, step := range stepGroup.Steps {
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
			scope[fmt.Sprintf("%s.status", prefix)] = true
			err := addItemsToScope(prefix, step.WithItems, step.WithParam, step.WithSequence, scope)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			err = validateArguments(fmt.Sprintf("templates.%s.steps[%d].%s.arguments.", tmpl.Name, i, step.Name), step.Arguments)
			if err != nil {
				return err
			}
			resolvedTmpl, err := ctx.validateTemplateHolder(&step, tmplCtx, &FakeArguments{})
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			resolvedTemplates[step.Name] = resolvedTmpl
		}

		stepBytes, err := json.Marshal(stepGroup)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		err = resolveAllVariables(scope, string(stepBytes))
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps %s", tmpl.Name, err.Error())
		}

		for _, step := range stepGroup.Steps {
			aggregate := len(step.WithItems) > 0 || step.WithParam != ""
			resolvedTmpl := resolvedTemplates[step.Name]
			ctx.addOutputsToScope(resolvedTmpl, fmt.Sprintf("steps.%s", step.Name), scope, aggregate, false)

			// Validate the template again with actual arguments.
			_, err = ctx.validateTemplateHolder(&step, tmplCtx, &step.Arguments)
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
			val := withItems[i]
			switch val.GetType() {
			case wfv1.String, wfv1.Number, wfv1.Bool:
				scope["item"] = true
			case wfv1.List:
				for i := range val.GetListVal() {
					scope[fmt.Sprintf("item.[%v]", i)] = true
				}
			case wfv1.Map:
				for itemKey := range val.GetMapVal() {
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
		if withSequence.Count != nil && withSequence.End != nil {
			return errors.New(errors.CodeBadRequest, "only one of count or end can be defined in withSequence")
		}
		scope["item"] = true
	}
	return nil
}

func (ctx *templateValidationCtx) addOutputsToScope(tmpl *wfv1.Template, prefix string, scope map[string]interface{}, aggregate bool, isAncestor bool) {
	if tmpl.Daemon != nil && *tmpl.Daemon {
		scope[fmt.Sprintf("%s.ip", prefix)] = true
	}
	if tmpl.Script != nil || tmpl.Container != nil {
		scope[fmt.Sprintf("%s.outputs.result", prefix)] = true
		scope[fmt.Sprintf("%s.exitCode", prefix)] = true
	}
	for _, param := range tmpl.Outputs.Parameters {
		scope[fmt.Sprintf("%s.outputs.parameters.%s", prefix, param.Name)] = true
		if param.GlobalName != "" {
			if !isParameter(param.GlobalName) {
				globalParamName := fmt.Sprintf("workflow.outputs.parameters.%s", param.GlobalName)
				scope[globalParamName] = true
				ctx.globalParams[globalParamName] = placeholderGenerator.NextPlaceholder()
			} else {
				logrus.Warnf("GlobalName '%s' is a parameter and won't be validated until runtime", param.GlobalName)
				scope[anyWorkflowOutputParameterMagicValue] = true
			}
		}
	}
	for _, art := range tmpl.Outputs.Artifacts {
		scope[fmt.Sprintf("%s.outputs.artifacts.%s", prefix, art.Name)] = true
		if art.GlobalName != "" {
			if !isParameter(art.GlobalName) {
				globalArtName := fmt.Sprintf("workflow.outputs.artifacts.%s", art.GlobalName)
				scope[globalArtName] = true
				ctx.globalParams[globalArtName] = placeholderGenerator.NextPlaceholder()
			} else {
				logrus.Warnf("GlobalName '%s' is a parameter and won't be validated until runtime", art.GlobalName)
				scope[anyWorkflowOutputArtifactMagicValue] = true
			}
		}
	}
	if aggregate {
		switch tmpl.GetType() {
		// Not that we don't also include TemplateTypeContainer here, even though it uses `outputs.result` it uses
		// `outputs.parameters` as its aggregator.
		case wfv1.TemplateTypeScript:
			scope[fmt.Sprintf("%s.outputs.result", prefix)] = true
			scope[fmt.Sprintf("%s.exitCode", prefix)] = true
		default:
			scope[fmt.Sprintf("%s.outputs.parameters", prefix)] = true
		}
	}
	if isAncestor {
		scope[fmt.Sprintf("%s.status", prefix)] = true
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
	// This validation is not applicable for DAG and Step Template types
	if tmpl.GetType() == wfv1.TemplateTypeDAG || tmpl.GetType() == wfv1.TemplateTypeSteps {
		return nil
	}
	switch ctx.ContainerRuntimeExecutor {
	case "", common.ContainerRuntimeExecutorDocker:
		// docker executor supports all modes of artifact outputs
	case common.ContainerRuntimeExecutorPNS:
		// pns supports copying from the base image, but only if there is no volume mount underneath it
		errMsg := "pns executor does not support outputs from base image layer with volume mounts. Use an emptyDir: https://argoproj.github.io/argo/empty-dir/"
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
		errMsg := fmt.Sprintf("%s executor does not support outputs from base image layer.  Use an emptyDir: https://argoproj.github.io/argo/empty-dir/", ctx.ContainerRuntimeExecutor)
		for _, out := range tmpl.Outputs.Artifacts {
			if common.FindOverlappingVolume(tmpl, out.Path) == nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs.artifacts.%s: %s", tmpl.Name, out.Name, errMsg)
			}
		}
		for _, out := range tmpl.Outputs.Parameters {
			if out.ValueFrom == nil {
				continue
			}
			if out.ValueFrom.Path != "" {
				if common.FindOverlappingVolume(tmpl, out.ValueFrom.Path) == nil {
					return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs.parameters.%s: %s", tmpl.Name, out.Name, errMsg)
				}
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
	if param.ValueFrom.Supplied != nil {
		paramTypes++
	}
	switch paramTypes {
	case 0:
		return errors.New(errors.CodeBadRequest, "valueFrom type unspecified. choose one of: path, jqFilter, jsonPath, parameter, raw")
	case 1:
	default:
		return errors.New(errors.CodeBadRequest, "multiple valueFrom types specified. choose one of: path, jqFilter, jsonPath, parameter, raw")
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

type dagValidationContext struct {
	tasks        map[string]wfv1.DAGTask
	dependencies map[string][]string
}

func (d *dagValidationContext) GetTask(taskName string) *wfv1.DAGTask {
	task := d.tasks[taskName]
	return &task
}

func (d *dagValidationContext) GetTaskDependencies(taskName string) []string {
	if dependencies, ok := d.dependencies[taskName]; ok {
		return dependencies
	}
	task := d.GetTask(taskName)
	dependencies, _ := common.GetTaskDependencies(task, d)
	d.dependencies[taskName] = dependencies
	return d.dependencies[taskName]
}

func (d *dagValidationContext) GetTaskFinishedAtTime(taskName string) time.Time {
	return time.Now()
}

func (ctx *templateValidationCtx) validateDAG(scope map[string]interface{}, tmplCtx *templateresolution.Context, tmpl *wfv1.Template) error {
	err := validateNonLeaf(tmpl)
	if err != nil {
		return err
	}
	if len(tmpl.DAG.Tasks) == 0 {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s must have at least one task", tmpl.Name)
	}

	err = sortDAGTasks(tmpl)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s sorting failed: %s", tmpl.Name, err.Error())
	}

	err = validateWorkflowFieldNames(tmpl.DAG.Tasks)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks%s", tmpl.Name, err.Error())
	}
	usingDepends := false
	nameToTask := make(map[string]wfv1.DAGTask)
	for _, task := range tmpl.DAG.Tasks {
		if task.Depends != "" {
			usingDepends = true
		}

		nameToTask[task.Name] = task
	}

	dagValidationCtx := &dagValidationContext{
		tasks:        nameToTask,
		dependencies: make(map[string][]string),
	}

	resolvedTemplates := make(map[string]*wfv1.Template)

	// Verify dependencies for all tasks can be resolved as well as template names
	for _, task := range tmpl.DAG.Tasks {

		if usingDepends && '0' <= task.Name[0] && task.Name[0] <= '9' {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s name cannot begin with a digit when using 'depends'", tmpl.Name, task.Name)
		}

		if usingDepends && len(task.Dependencies) > 0 {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s cannot use both 'depends' and 'dependencies' in the same DAG template", tmpl.Name)
		}

		if usingDepends && task.ContinueOn != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s cannot use 'continueOn' when using 'depends'. Instead use 'dep-task.Failed'/'dep-task.Errored'", tmpl.Name)
		}

		resolvedTmpl, err := ctx.validateTemplateHolder(&task, tmplCtx, &FakeArguments{})
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}

		resolvedTemplates[task.Name] = resolvedTmpl

		prefix := fmt.Sprintf("tasks.%s", task.Name)
		ctx.addOutputsToScope(resolvedTmpl, prefix, scope, false, false)

		err = common.ValidateTaskResults(&task)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}

		for j, depName := range dagValidationCtx.GetTaskDependencies(task.Name) {
			if _, ok := dagValidationCtx.tasks[depName]; !ok {
				return errors.Errorf(errors.CodeBadRequest,
					"templates.%s.tasks.%s.dependencies[%d] dependency '%s' not defined",
					tmpl.Name, task.Name, j, depName)
			}
		}
	}

	if err = verifyNoCycles(tmpl, dagValidationCtx); err != nil {
		return err
	}

	err = resolveAllVariables(scope, tmpl.DAG.Target)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.targets %s", tmpl.Name, err.Error())
	}
	if err = validateDAGTargets(tmpl, dagValidationCtx.tasks); err != nil {
		return err
	}

	for _, task := range tmpl.DAG.Tasks {
		resolvedTmpl := resolvedTemplates[task.Name]
		// add all tasks outputs to scope so that a nested DAGs can have outputs
		prefix := fmt.Sprintf("tasks.%s", task.Name)
		ctx.addOutputsToScope(resolvedTmpl, prefix, scope, false, false)
		taskBytes, err := json.Marshal(task)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		taskScope := make(map[string]interface{})
		for k, v := range scope {
			taskScope[k] = v
		}
		ancestry := common.GetTaskAncestry(dagValidationCtx, task.Name)
		for _, ancestor := range ancestry {
			ancestorTask := dagValidationCtx.GetTask(ancestor)
			resolvedTmpl := resolvedTemplates[ancestor]
			ancestorPrefix := fmt.Sprintf("tasks.%s", ancestor)
			aggregate := len(ancestorTask.WithItems) > 0 || ancestorTask.WithParam != ""
			ctx.addOutputsToScope(resolvedTmpl, ancestorPrefix, taskScope, aggregate, true)
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
		_, err = ctx.validateTemplateHolder(&task, tmplCtx, &task.Arguments)
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
func verifyNoCycles(tmpl *wfv1.Template, ctx *dagValidationContext) error {
	visited := make(map[string]bool)
	var noCyclesHelper func(taskName string, cycle []string) error
	noCyclesHelper = func(taskName string, cycle []string) error {
		if _, ok := visited[taskName]; ok {
			return nil
		}
		task := ctx.GetTask(taskName)
		for _, depName := range ctx.GetTaskDependencies(task.Name) {
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

func sortDAGTasks(tmpl *wfv1.Template) error {
	taskMap := make(map[string]*wfv1.DAGTask, len(tmpl.DAG.Tasks))
	sortingGraph := make([]*sorting.TopologicalSortingNode, len(tmpl.DAG.Tasks))
	for index := range tmpl.DAG.Tasks {
		taskMap[tmpl.DAG.Tasks[index].Name] = &tmpl.DAG.Tasks[index]
		sortingGraph[index] = &sorting.TopologicalSortingNode{
			NodeName:     tmpl.DAG.Tasks[index].Name,
			Dependencies: tmpl.DAG.Tasks[index].Dependencies,
		}
	}
	sortingResult, err := sorting.TopologicalSorting(sortingGraph)
	if err != nil {
		return err
	}
	tmpl.DAG.Tasks = make([]wfv1.DAGTask, len(tmpl.DAG.Tasks))
	for index, node := range sortingResult {
		tmpl.DAG.Tasks[index] = *taskMap[node.NodeName]
	}
	return nil
}

var (
	// paramRegex matches a parameter. e.g. {{inputs.parameters.blah}}
	paramRegex               = regexp.MustCompile(`{{[-a-zA-Z0-9]+(\.[-a-zA-Z0-9_]+)*}}`)
	paramOrArtifactNameRegex = regexp.MustCompile(`^[-a-zA-Z0-9_]+[-a-zA-Z0-9_]*$`)
	workflowFieldNameRegex   = regexp.MustCompile("^" + workflowFieldNameFmt + "$")
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

const (
	workflowFieldNameFmt    string = "[a-zA-Z0-9][-a-zA-Z0-9]*"
	workflowFieldNameErrMsg string = "name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character"
	workflowFieldMaxLength  int    = 128
)

// isValidWorkflowFieldName : workflow field name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character
func isValidWorkflowFieldName(name string) []string {
	var errs []string
	if len(name) > workflowFieldMaxLength {
		errs = append(errs, apivalidation.MaxLenError(workflowFieldMaxLength))
	}
	if !workflowFieldNameRegex.MatchString(name) {
		msg := workflowFieldNameErrMsg + " (e.g. My-name1-2, 123-NAME)"
		errs = append(errs, msg)
	}
	return errs
}

func getTemplateID(tmpl *wfv1.Template) string {
	return fmt.Sprintf("%s %v", tmpl.Name, tmpl.TemplateRef)
}
