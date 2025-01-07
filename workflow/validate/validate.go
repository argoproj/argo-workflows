package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apivalidation "k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/intstr"
	"github.com/argoproj/argo-workflows/v3/util/sorting"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/hdfs"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

// ValidateOpts provides options when linting
type ValidateOpts struct {
	// Lint indicates if this is performing validation in the context of linting. If true, will
	// skip some validations which is permissible during linting but not submission (e.g. missing
	// input parameters to the workflow)
	Lint bool

	// IgnoreEntrypoint indicates to skip/ignore the EntryPoint validation on workflow spec.
	// Entrypoint is optional for WorkflowTemplate and ClusterWorkflowTemplate
	IgnoreEntrypoint bool

	// WorkflowTemplateValidation indicates that the current context is validating a WorkflowTemplate or ClusterWorkflowTemplate
	WorkflowTemplateValidation bool

	// Submit indicates that the current operation is a workflow submission. This will impose
	// more stringent requirements (e.g. require input values for all spec arguments)
	Submit bool
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
	globalParams[common.GlobalVarWorkflowMainEntrypoint] = placeholderGenerator.NextPlaceholder()
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
	// The maximum length of maxCharsInObjectName is 63 characters because of the limitation of Kubernetes label
	// For details, please refer to: https://stackoverflow.com/questions/50412837/kubernetes-label-name-63-character-limit
	maxCharsInObjectName = 63
	// CronWorkflows have fewer max chars allowed in their name because when workflows are created from them, they
	// are appended with the unix timestamp (`-1615836720`). This lower character allowance allows for that timestamp
	// to still fit within the 63 character maximum.
	maxCharsInCronWorkflowName = 52
)

var placeholderGenerator = common.NewPlaceholderGenerator()

type FakeArguments struct{}

func (args *FakeArguments) GetParameterByName(name string) *wfv1.Parameter {
	s := placeholderGenerator.NextPlaceholder()
	return &wfv1.Parameter{Name: name, Value: wfv1.AnyStringPtr(s)}
}

func (args *FakeArguments) GetArtifactByName(name string) *wfv1.Artifact {
	return &wfv1.Artifact{Name: name}
}

var _ wfv1.ArgumentsProvider = &FakeArguments{}

func SubstituteResourceManifestExpressions(manifest string) string {
	var substitutions = make(map[string]string)
	pattern, _ := regexp.Compile(`{{\s*=\s*(.+?)\s*}}`)
	for _, match := range pattern.FindAllStringSubmatch(manifest, -1) {
		substitutions[string(match[1])] = placeholderGenerator.NextPlaceholder()
	}

	// since we don't need to resolve/evaluate here we can do just a simple replacement
	for old, new := range substitutions {
		rmatch, _ := regexp.Compile(`{{\s*=\s*` + regexp.QuoteMeta(old) + `\s*}}`)
		manifest = rmatch.ReplaceAllString(manifest, new)
	}

	return manifest
}

// validateHooks takes an array of hooks to validate and the name of the
// container they are in and generates an error for the first invalid hook
// or nil if they are all valid
func validateHooks(hooks wfv1.LifecycleHooks, hookBaseName string) error {
	for hookName, hook := range hooks {
		if hookName != wfv1.ExitLifecycleEvent && hook.Expression == "" {
			return errors.Errorf(errors.CodeBadRequest, "%s.%s %s", hookBaseName, hookName, "Expression required")
		}
	}
	return nil
}

// ValidateWorkflow accepts a workflow and performs validation against it.
func ValidateWorkflow(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, wf *wfv1.Workflow, wfDefaults *wfv1.Workflow, opts ValidateOpts) error {
	ctx := newTemplateValidationCtx(wf, opts)
	tmplCtx := templateresolution.NewContext(wftmplGetter, cwftmplGetter, wf, wf)
	var wfSpecHolder wfv1.WorkflowSpecHolder
	var wfTmplRef *wfv1.TemplateRef
	var err error

	if len(wf.Name) > maxCharsInObjectName {
		return fmt.Errorf("workflow name %q must not be more than 63 characters long (currently %d)", wf.Name, len(wf.Name))
	}

	entrypoint := wf.Spec.Entrypoint

	hasWorkflowTemplateRef := wf.Spec.WorkflowTemplateRef != nil

	if hasWorkflowTemplateRef {
		err := ValidateWorkflowTemplateRefFields(wf.Spec)
		if err != nil {
			return err
		}
		if wf.Spec.WorkflowTemplateRef.ClusterScope {
			wfSpecHolder, err = cwftmplGetter.Get(wf.Spec.WorkflowTemplateRef.Name)
		} else {
			wfSpecHolder, err = wftmplGetter.Get(wf.Spec.WorkflowTemplateRef.Name)
		}
		if err != nil {
			return err
		}
		if entrypoint == "" {
			entrypoint = wfSpecHolder.GetWorkflowSpec().Entrypoint
		}
		wfTmplRef = wf.Spec.WorkflowTemplateRef.ToTemplateRef(entrypoint)
	}
	err = validateWorkflowFieldNames(wf.Spec.Templates)

	wfArgs := wf.Spec.Arguments

	if hasWorkflowTemplateRef {
		wfArgs.Parameters = util.MergeParameters(wfArgs.Parameters, wfSpecHolder.GetWorkflowSpec().Arguments.Parameters)
		wfArgs.Artifacts = util.MergeArtifacts(wfArgs.Artifacts, wfSpecHolder.GetWorkflowSpec().Arguments.Artifacts)
	}
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "spec.templates%s", err.Error())
	}

	// if we are linting, we don't care if spec.arguments.parameters.XXX doesn't have an
	// explicit value. Workflow templates without a default value are also a desired use
	// case, since values will be provided during workflow submission.
	allowEmptyValues := ctx.Lint || (ctx.WorkflowTemplateValidation && !ctx.Submit)
	err = validateArguments("spec.arguments.", wfArgs, allowEmptyValues)
	if err != nil {
		return err
	}
	if len(wfArgs.Parameters) > 0 {
		ctx.globalParams[common.GlobalVarWorkflowParameters] = placeholderGenerator.NextPlaceholder()
		ctx.globalParams[common.GlobalVarWorkflowParametersJSON] = placeholderGenerator.NextPlaceholder()
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

	annotationSources := [][]string{maps.Keys(wf.ObjectMeta.Annotations)}
	labelSources := [][]string{maps.Keys(wf.ObjectMeta.Labels)}
	if wf.Spec.WorkflowMetadata != nil {
		annotationSources = append(annotationSources, maps.Keys(wf.Spec.WorkflowMetadata.Annotations))
		labelSources = append(labelSources, maps.Keys(wf.Spec.WorkflowMetadata.Labels), maps.Keys(wf.Spec.WorkflowMetadata.LabelsFrom))
	}
	if wfDefaults != nil && wfDefaults.Spec.WorkflowMetadata != nil {
		annotationSources = append(annotationSources, maps.Keys(wfDefaults.Spec.WorkflowMetadata.Annotations))
		labelSources = append(labelSources, maps.Keys(wfDefaults.Spec.WorkflowMetadata.Labels), maps.Keys(wfDefaults.Spec.WorkflowMetadata.LabelsFrom))
	}
	if wf.Spec.WorkflowTemplateRef != nil && wfSpecHolder.GetWorkflowSpec().WorkflowMetadata != nil {
		annotationSources = append(annotationSources, maps.Keys(wfSpecHolder.GetWorkflowSpec().WorkflowMetadata.Annotations))
		labelSources = append(labelSources, maps.Keys(wfSpecHolder.GetWorkflowSpec().WorkflowMetadata.Labels), maps.Keys(wfSpecHolder.GetWorkflowSpec().WorkflowMetadata.LabelsFrom))
	}
	mergedAnnotations := getUniqueKeys(annotationSources...)
	mergedLabels := getUniqueKeys(labelSources...)

	for k := range mergedAnnotations {
		ctx.globalParams["workflow.annotations."+k] = placeholderGenerator.NextPlaceholder()
	}
	ctx.globalParams[common.GlobalVarWorkflowAnnotations] = placeholderGenerator.NextPlaceholder()
	ctx.globalParams[common.GlobalVarWorkflowAnnotationsJSON] = placeholderGenerator.NextPlaceholder()

	for k := range mergedLabels {
		ctx.globalParams["workflow.labels."+k] = placeholderGenerator.NextPlaceholder()
	}
	ctx.globalParams[common.GlobalVarWorkflowLabels] = placeholderGenerator.NextPlaceholder()
	ctx.globalParams[common.GlobalVarWorkflowLabelsJSON] = placeholderGenerator.NextPlaceholder()

	if wf.Spec.Priority != nil {
		ctx.globalParams[common.GlobalVarWorkflowPriority] = strconv.Itoa(int(*wf.Spec.Priority))
	}
	ctx.globalParams[common.GlobalVarWorkflowStatus] = placeholderGenerator.NextPlaceholder()

	if !opts.IgnoreEntrypoint && entrypoint == "" {
		return errors.New(errors.CodeBadRequest, "spec.entrypoint is required")
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
		_, err = ctx.validateTemplateHolder(tmpl, tmplCtx, args, opts.WorkflowTemplateValidation)
		if err != nil {
			return err
		}
	}

	// Validate OnExit hooks
	// If the OnExit is specified in Workflow:
	//   - If the template that referred by OnExit is from a WorkflowTemplate, template in WorkflowTemplate will be validated.
	//   - If the template is inlined, the template will be validated.
	// If the OnExit is empty in Workflow:
	//   - If OnExit is specified in referred WorkflowTemplate, the OnExit of the referred WorkflowTemplate will be validated.
	//   - If OnExit is empty in referred WorkflowTemplate, nothing will be validated.
	var tmplHolder *wfv1.WorkflowStep
	if wf.Spec.OnExit != "" {
		tmplHolder = &wfv1.WorkflowStep{Template: wf.Spec.OnExit}
		if hasWorkflowTemplateRef {
			tmplHolder = &wfv1.WorkflowStep{TemplateRef: wf.Spec.WorkflowTemplateRef.ToTemplateRef(wf.Spec.OnExit)}
		}
	} else if hasWorkflowTemplateRef && wfSpecHolder.GetWorkflowSpec().OnExit != "" {
		tmplHolder = &wfv1.WorkflowStep{TemplateRef: wf.Spec.WorkflowTemplateRef.ToTemplateRef(wfSpecHolder.GetWorkflowSpec().OnExit)}
	}
	if tmplHolder != nil {
		ctx.globalParams[common.GlobalVarWorkflowFailures] = placeholderGenerator.NextPlaceholder()
		_, err = ctx.validateTemplateHolder(tmplHolder, tmplCtx, &wf.Spec.Arguments, opts.WorkflowTemplateValidation)
		if err != nil {
			return err
		}
	}
	err = validateHooks(wf.Spec.Hooks, "hooks")
	if err != nil {
		return err
	}

	if !wf.Spec.PodGC.GetStrategy().IsValid() {
		return errors.Errorf(errors.CodeBadRequest, "podGC.strategy unknown strategy '%s'", wf.Spec.PodGC.Strategy)
	}
	if _, err := wf.Spec.PodGC.GetLabelSelector(); err != nil {
		return errors.Errorf(errors.CodeBadRequest, "podGC.labelSelector invalid: %v", err)
	}

	// Check if all templates can be resolved.
	// If the Workflow is using a WorkflowTemplateRef, then the templates of the referred WorkflowTemplate will be validated.
	if hasWorkflowTemplateRef {
		for _, template := range wfSpecHolder.GetWorkflowSpec().Templates {
			_, err := ctx.validateTemplateHolder(&wfv1.WorkflowStep{TemplateRef: wf.Spec.WorkflowTemplateRef.ToTemplateRef(template.Name)}, tmplCtx, &FakeArguments{}, opts.WorkflowTemplateValidation)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s %s", template.Name, err.Error())
			}
		}
		return nil
	}
	// If the templates are inlined in Workflow, then the inlined templates will be validated.
	for _, template := range wf.Spec.Templates {
		_, err := ctx.validateTemplateHolder(&wfv1.WorkflowStep{Template: template.Name}, tmplCtx, &FakeArguments{}, opts.WorkflowTemplateValidation)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s %s", template.Name, err.Error())
		}
	}
	return nil
}

// construct a Set of unique keys
func getUniqueKeys(labelSources ...[]string) map[string]struct{} {
	uniqueKeys := make(map[string]struct{})
	for _, labelSource := range labelSources {
		for _, label := range labelSource {
			uniqueKeys[label] = struct{}{} // dummy value
		}
	}
	return uniqueKeys
}

func ValidateWorkflowTemplateRefFields(wfSpec wfv1.WorkflowSpec) error {
	if len(wfSpec.Templates) > 0 {
		return errors.Errorf(errors.CodeBadRequest, "Templates is invalid field in spec if workflow referred WorkflowTemplate reference")
	}
	return nil
}

// ValidateWorkflowTemplate accepts a workflow template and performs validation against it.
func ValidateWorkflowTemplate(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, wftmpl *wfv1.WorkflowTemplate, wfDefaults *wfv1.Workflow, opts ValidateOpts) error {
	if len(wftmpl.Name) > maxCharsInObjectName {
		return fmt.Errorf("workflow template name %q must not be more than 63 characters long (currently %d)", wftmpl.Name, len(wftmpl.Name))
	}

	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Labels:      wftmpl.ObjectMeta.Labels,
			Annotations: wftmpl.ObjectMeta.Annotations,
		},
		Spec: wftmpl.Spec,
	}
	opts.IgnoreEntrypoint = wf.Spec.Entrypoint == ""
	opts.WorkflowTemplateValidation = true
	return ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, wfDefaults, opts)
}

// ValidateClusterWorkflowTemplate accepts a cluster workflow template and performs validation against it.
func ValidateClusterWorkflowTemplate(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, cwftmpl *wfv1.ClusterWorkflowTemplate, wfDefaults *wfv1.Workflow, opts ValidateOpts) error {
	if len(cwftmpl.Name) > maxCharsInObjectName {
		return fmt.Errorf("cluster workflow template name %q must not be more than 63 characters long (currently %d)", cwftmpl.Name, len(cwftmpl.Name))
	}

	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Labels:      cwftmpl.ObjectMeta.Labels,
			Annotations: cwftmpl.ObjectMeta.Annotations,
		},
		Spec: cwftmpl.Spec,
	}
	opts.IgnoreEntrypoint = wf.Spec.Entrypoint == ""
	opts.WorkflowTemplateValidation = true
	return ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, wfDefaults, opts)
}

// ValidateCronWorkflow validates a CronWorkflow
func ValidateCronWorkflow(ctx context.Context, wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, cronWf *wfv1.CronWorkflow, wfDefaults *wfv1.Workflow) error {
	if len(cronWf.Spec.Schedules) > 0 && cronWf.Spec.Schedule != "" {
		return fmt.Errorf("cron workflow cant be configured with both Spec.Schedule and Spec.Schedules")
	}
	// CronWorkflows have fewer max chars allowed in their name because when workflows are created from them, they
	// are appended with the unix timestamp (`-1615836720`). This lower character allowance allows for that timestamp
	// to still fit within the 63 character maximum.
	if len(cronWf.Name) > maxCharsInCronWorkflowName {
		return fmt.Errorf("cron workflow name %q must not be more than 52 characters long (currently %d)", cronWf.Name, len(cronWf.Name))
	}

	for _, schedule := range cronWf.Spec.GetSchedules(ctx) {
		if _, err := cron.ParseStandard(schedule); err != nil {
			return errors.Errorf(errors.CodeBadRequest, "cron schedule %s is malformed: %s", schedule, err)
		}
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

	err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, wfDefaults, ValidateOpts{})
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "cannot validate Workflow: %s", err)
	}
	return nil
}

func (ctx *templateValidationCtx) validateInitContainers(containers []wfv1.UserContainer) error {
	for _, container := range containers {
		if len(container.Container.Name) == 0 {
			return errors.Errorf(errors.CodeBadRequest, "initContainers must all have container name")
		}
	}
	return nil
}

func (ctx *templateValidationCtx) validateTemplate(tmpl *wfv1.Template, tmplCtx *templateresolution.Context, args wfv1.ArgumentsProvider, workflowTemplateValidation bool) error {

	if err := validateTemplateType(tmpl); err != nil {
		return err
	}

	scope, err := validateInputs(tmpl)
	if err != nil {
		return err
	}

	if err := ctx.validateInitContainers(tmpl.InitContainers); err != nil {
		return err
	}

	localParams := make(map[string]string)
	if tmpl.IsPodType() {
		localParams[common.LocalVarPodName] = placeholderGenerator.NextPlaceholder()
		scope[common.LocalVarPodName] = placeholderGenerator.NextPlaceholder()
	}
	if tmpl.RetryStrategy != nil {
		localParams[common.LocalVarRetries] = placeholderGenerator.NextPlaceholder()
		localParams[common.LocalVarRetriesLastExitCode] = placeholderGenerator.NextPlaceholder()
		localParams[common.LocalVarRetriesLastStatus] = placeholderGenerator.NextPlaceholder()
		localParams[common.LocalVarRetriesLastDuration] = placeholderGenerator.NextPlaceholder()
		localParams[common.LocalVarRetriesLastMessage] = placeholderGenerator.NextPlaceholder()
		scope[common.LocalVarRetries] = placeholderGenerator.NextPlaceholder()
		scope[common.LocalVarRetriesLastExitCode] = placeholderGenerator.NextPlaceholder()
		scope[common.LocalVarRetriesLastStatus] = placeholderGenerator.NextPlaceholder()
		scope[common.LocalVarRetriesLastDuration] = placeholderGenerator.NextPlaceholder()
		scope[common.LocalVarRetriesLastMessage] = placeholderGenerator.NextPlaceholder()
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

	newTmpl, err := common.ProcessArgs(tmpl, args, ctx.globalParams, localParams, true, "", nil)
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

	templateScope := tmplCtx.GetTemplateScope()
	tmplID := getTemplateID(tmpl)
	_, ok := ctx.results[templateScope+tmplID]
	if ok {
		// we can skip the rest since it has been validated.
		return nil
	}
	ctx.results[templateScope+tmplID] = true

	for globalVar, val := range ctx.globalParams {
		scope[globalVar] = val
	}
	switch newTmpl.GetType() {
	case wfv1.TemplateTypeSteps:
		err = ctx.validateSteps(scope, tmplCtx, newTmpl, workflowTemplateValidation)
	case wfv1.TemplateTypeDAG:
		err = ctx.validateDAG(scope, tmplCtx, newTmpl, workflowTemplateValidation)
	default:
		err = ctx.validateLeaf(scope, tmplCtx, newTmpl, workflowTemplateValidation)
	}
	if err != nil {
		return err
	}
	err = validateOutputs(scope, ctx.globalParams, newTmpl, workflowTemplateValidation)
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
				return errors.Errorf(errors.CodeBadRequest, "templates.%s metric name '%s' is invalid. Metric names must contain alphanumeric characters or '_'", tmpl.Name, metric.Name)
			}
			if err := metrics.ValidateMetricLabels(metric.GetMetricLabels()); err != nil {
				return err
			}
			if metric.Help == "" {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s metric '%s' must contain a help string under 'help: ' field", tmpl.Name, metric.Name)
			}
			if err := metrics.ValidateMetricValues(metric); err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s metric '%s' error: %s", tmpl.Name, metric.Name, err)
			}
		}
	}
	return nil
}

// VerifyResolvedVariables is a helper to ensure all {{variables}} have been resolved for a object
func VerifyResolvedVariables(obj interface{}) error {
	str, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return template.Validate(string(str), func(tag string) error {
		return errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}}", tag)
	})
}

// validateTemplateHolder validates a template holder and returns the validated template.
func (ctx *templateValidationCtx) validateTemplateHolder(tmplHolder wfv1.TemplateReferenceHolder, tmplCtx *templateresolution.Context, args wfv1.ArgumentsProvider, workflowTemplateValidation bool) (*wfv1.Template, error) {
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
		if err := VerifyResolvedVariables(tmplRef); err != nil {
			logrus.Warnf("template reference need resolution: %v", err)
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
		case wfv1.RetryPolicyAlways, wfv1.RetryPolicyOnError, wfv1.RetryPolicyOnFailure, wfv1.RetryPolicyOnTransientError, "":
			// Passes validation
		default:
			return nil, fmt.Errorf("%s is not a valid RetryPolicy", resolvedTmpl.RetryStrategy.RetryPolicy)
		}
	}

	return resolvedTmpl, ctx.validateTemplate(resolvedTmpl, tmplCtx, args, workflowTemplateValidation)
}

// validateTemplateType validates that only one template type is defined
func validateTemplateType(tmpl *wfv1.Template) error {
	numTypes := 0
	for _, tmplType := range []interface{}{tmpl.Container, tmpl.ContainerSet, tmpl.Steps, tmpl.Script, tmpl.Resource, tmpl.DAG, tmpl.Suspend, tmpl.Data, tmpl.HTTP, tmpl.Plugin} {
		if !reflect.ValueOf(tmplType).IsNil() {
			numTypes++
		}
	}
	switch numTypes {
	case 0:
		return errors.Errorf(errors.CodeBadRequest, "templates.%s template type unspecified. choose one of: container, containerSet, steps, script, resource, dag, suspend, template, template ref", tmpl.Name)
	case 1:
		// Do nothing
	default:
		return errors.Errorf(errors.CodeBadRequest, "templates.%s multiple template types specified. choose one of: container, containerSet, steps, script, resource, dag, suspend, template, template ref", tmpl.Name)
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
			err = art.CleanPath()
			if err != nil {
				return nil, errors.Errorf(errors.CodeBadRequest, "error in templates.%s.%s: %s", tmpl.Name, artRef, err.Error())
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

// resolveAllVariables is a helper to ensure all {{variables}} are resolvable from current scope
func resolveAllVariables(scope map[string]interface{}, globalParams map[string]string, tmplStr string, workflowTemplateValidation bool) error {
	_, allowAllItemRefs := scope[anyItemMagicValue] // 'item.*' is a magic placeholder value set by addItemsToScope
	_, allowAllWorkflowOutputParameterRefs := scope[anyWorkflowOutputParameterMagicValue]
	_, allowAllWorkflowOutputArtifactRefs := scope[anyWorkflowOutputArtifactMagicValue]
	return template.Validate(tmplStr, func(tag string) error {
		// Trim the tag to check the validations
		trimmedTag := strings.TrimSpace(tag)
		// Skip the custom variable references
		if !checkValidWorkflowVariablePrefix(trimmedTag) {
			return nil
		}
		_, ok := scope[trimmedTag]
		_, isGlobal := globalParams[trimmedTag]
		if !ok && !isGlobal {
			if (trimmedTag == "item" || strings.HasPrefix(trimmedTag, "item.")) && allowAllItemRefs {
				// we are *probably* referencing a undetermined item using withParam
				// NOTE: this is far from foolproof.
			} else if strings.HasPrefix(trimmedTag, "workflow.outputs.parameters.") && allowAllWorkflowOutputParameterRefs {
				// Allow runtime resolution of workflow output parameter names
			} else if strings.HasPrefix(trimmedTag, "workflow.outputs.artifacts.") && allowAllWorkflowOutputArtifactRefs {
				// Allow runtime resolution of workflow output artifact names
			} else if strings.HasPrefix(trimmedTag, "outputs.") {
				// We are self referencing for metric emission, allow it.
			} else if strings.HasPrefix(trimmedTag, common.GlobalVarWorkflowCreationTimestamp) {
			} else if strings.HasPrefix(trimmedTag, common.GlobalVarWorkflowCronScheduleTime) {
				// Allow runtime resolution for "scheduledTime" which will pass from CronWorkflow
			} else if strings.HasPrefix(trimmedTag, common.GlobalVarWorkflowDuration) {
			} else if strings.HasPrefix(trimmedTag, "tasks.name") {
			} else if strings.HasPrefix(trimmedTag, "steps.name") {
			} else if strings.HasPrefix(trimmedTag, "node.name") {
			} else if strings.HasPrefix(trimmedTag, "workflow.parameters") && workflowTemplateValidation {
				// If we are simply validating a WorkflowTemplate in isolation, some of the parameters may come from the Workflow that uses it
			} else {
				return fmt.Errorf("failed to resolve {{%s}}", tag)
			}
		}
		return nil
	})
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

func (ctx *templateValidationCtx) validateLeaf(scope map[string]interface{}, tmplCtx *templateresolution.Context, tmpl *wfv1.Template, workflowTemplateValidation bool) error {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	err = resolveAllVariables(scope, ctx.globalParams, string(tmplBytes), workflowTemplateValidation)
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
			switch baseTemplate := tmplCtx.GetCurrentTemplateBase().(type) {
			case *wfv1.Workflow:
				if !(baseTemplate.Spec.TemplateDefaults != nil && baseTemplate.Spec.TemplateDefaults.Container != nil && baseTemplate.Spec.TemplateDefaults.Container.Image != "") {
					return errors.Errorf(errors.CodeBadRequest, "templates.%s.container.image may not be empty", tmpl.Name)
				}
			case *wfv1.WorkflowTemplate:
				if !(baseTemplate.Spec.TemplateDefaults != nil && baseTemplate.Spec.TemplateDefaults.Container != nil && baseTemplate.Spec.TemplateDefaults.Container.Image != "") {
					return errors.Errorf(errors.CodeBadRequest, "templates.%s.container.image may not be empty", tmpl.Name)
				}
			default:
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.container.image may not be empty", tmpl.Name)
			}
		}
	}
	if tmpl.ContainerSet != nil {
		err = tmpl.ContainerSet.Validate()
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.containerSet.%s", tmpl.Name, err.Error())
		}
		if len(tmpl.Inputs.Artifacts) > 0 || len(tmpl.Outputs.Parameters) > 0 || len(tmpl.Outputs.Artifacts) > 0 {
			if !tmpl.ContainerSet.HasContainerNamed("main") {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.containerSet.containers must have a container named \"main\" for input or output", tmpl.Name)
			}
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
		if tmpl.Resource.Action != "delete" && tmpl.Resource.Action != "get" {
			if tmpl.Resource.Manifest == "" && tmpl.Resource.ManifestFrom == nil {
				return errors.Errorf(errors.CodeBadRequest, "either templates.%s.resource.manifest or templates.%s.resource.manifestFrom must be specified", tmpl.Name, tmpl.Name)
			}
			if tmpl.Resource.Manifest != "" && tmpl.Resource.ManifestFrom != nil {
				return errors.Errorf(errors.CodeBadRequest, "shouldn't have both `manifest` and `manifestFrom` specified in `Manifest` for resource template")
			}
			if tmpl.Resource.ManifestFrom != nil && tmpl.Resource.ManifestFrom.Artifact != nil {
				var found bool
				for _, art := range tmpl.Inputs.Artifacts {
					if tmpl.Resource.ManifestFrom.Artifact.Name == art.Name {
						found = true
						break
					}
				}
				if !found {
					return errors.Errorf(errors.CodeBadRequest, "artifact %s in `manifestFrom` refer to a non-exist artifact", tmpl.Resource.ManifestFrom.Artifact.Name)
				}
			}
			if tmpl.Resource.Manifest != "" && !placeholderGenerator.IsPlaceholder(tmpl.Resource.Manifest) {
				// Try to unmarshal the given manifest, just ensuring it's a valid YAML.
				var obj interface{}

				// Unmarshalling will fail if we have unquoted expressions which is sometimes a false positive,
				// so for the sake of template validation we will just replace expressions with placeholders
				// and the 'real' validation will be performed at a later stage once the manifest is applied
				replaced := SubstituteResourceManifestExpressions(tmpl.Resource.Manifest)
				err := yaml.Unmarshal([]byte(replaced), &obj)
				if err != nil {
					return errors.Errorf(errors.CodeBadRequest, "templates.%s.resource.manifest must be a valid yaml", tmpl.Name)
				}
			}
		}
	}
	if tmpl.Script != nil {
		if tmpl.Script.Image == "" {
			switch baseTemplate := tmplCtx.GetCurrentTemplateBase().(type) {
			case *wfv1.Workflow:
				if !(baseTemplate.Spec.TemplateDefaults != nil && baseTemplate.Spec.TemplateDefaults.Script != nil && baseTemplate.Spec.TemplateDefaults.Script.Image != "") {
					return errors.Errorf(errors.CodeBadRequest, "templates.%s.script.image may not be empty", tmpl.Name)
				}
			case *wfv1.WorkflowTemplate:
				if !(baseTemplate.Spec.TemplateDefaults != nil && baseTemplate.Spec.TemplateDefaults.Script != nil && baseTemplate.Spec.TemplateDefaults.Script.Image != "") {
					return errors.Errorf(errors.CodeBadRequest, "templates.%s.script.image may not be empty", tmpl.Name)
				}
			default:
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.script.image may not be empty", tmpl.Name)
			}
		}
	}
	// we don't validate tmpl.Plugin, because this is done by Plugin.UnmarshallJSON
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
	return nil
}

func validateArguments(prefix string, arguments wfv1.Arguments, allowEmptyValues bool) error {
	err := validateArgumentsFieldNames(prefix, arguments)
	if err != nil {
		return err
	}
	return validateArgumentsValues(prefix, arguments, allowEmptyValues)
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
func validateArgumentsValues(prefix string, arguments wfv1.Arguments, allowEmptyValues bool) error {
	for _, param := range arguments.Parameters {
		// check if any value is defined
		if param.ValueFrom == nil && param.Value == nil {
			if !allowEmptyValues {
				return errors.Errorf(errors.CodeBadRequest, "%s%s.value or %s%s.valueFrom is required", prefix, param.Name, prefix, param.Name)
			}
		}
		if param.ValueFrom != nil {
			// check for valid valueFrom sub-parameters
			// INFO: default needs to be accompanied by ConfigMapKeyRef.
			if param.ValueFrom.ConfigMapKeyRef == nil && param.ValueFrom.Event == "" && param.ValueFrom.Supplied == nil {
				return errors.Errorf(errors.CodeBadRequest, "%s%s.valueFrom only allows: default, configMapKeyRef and supplied", prefix, param.Name)
			}
			// check for invalid valueFrom sub-parameters
			if param.ValueFrom.Path != "" || param.ValueFrom.JSONPath != "" || param.ValueFrom.Parameter != "" || param.ValueFrom.Expression != "" {
				return errors.Errorf(errors.CodeBadRequest, "%s%s.valueFrom only allows: default, configMapKeyRef and supplied", prefix, param.Name)
			}
		}
		// validate enum
		if param.Enum != nil {
			if len(param.Enum) == 0 {
				return errors.Errorf(errors.CodeBadRequest, "%s%s.enum should contain at least one value", prefix, param.Name)
			}
			if param.Value == nil {
				if allowEmptyValues {
					return nil
				}
				return errors.Errorf(errors.CodeBadRequest, "%s%s.value is required", prefix, param.Name)
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
		if art.From == "" && !art.HasLocationOrKey() {
			return errors.Errorf(errors.CodeBadRequest, "%s%s.from, artifact location, or key is required", prefix, art.Name)
		}
		if art.From != "" && art.FromExpression != "" {
			return errors.Errorf(errors.CodeBadRequest, "%s%s shouldn't have both `from` and `fromExpression` in Artifact", prefix, art.Name)
		}
	}
	return nil
}

func (ctx *templateValidationCtx) validateSteps(scope map[string]interface{}, tmplCtx *templateresolution.Context, tmpl *wfv1.Template, workflowTemplateValidation bool) error {
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
			err := addItemsToScope(step.WithItems, step.WithParam, step.WithSequence, scope)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			err = validateArguments(fmt.Sprintf("templates.%s.steps[%d].%s.arguments.", tmpl.Name, i, step.Name), step.Arguments, false)
			if err != nil {
				return err
			}
			var args wfv1.ArgumentsProvider
			args = &FakeArguments{}
			if step.TemplateRef != nil {
				args = &step.Arguments
			}
			resolvedTmpl, err := ctx.validateTemplateHolder(&step, tmplCtx, args, workflowTemplateValidation)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}

			if step.HasExitHook() {
				ctx.addOutputsToScope(resolvedTmpl, fmt.Sprintf("steps.%s", step.Name), scope, false, false)
			}
			resolvedTemplates[step.Name] = resolvedTmpl

			err = validateHooks(step.Hooks, fmt.Sprintf("templates.%s.steps[%d].%s", tmpl.Name, i, step.Name))
			if err != nil {
				return err
			}

			stepBytes, err := json.Marshal(step)
			if err != nil {
				return errors.InternalWrapError(err)
			}

			stepScope := make(map[string]interface{})
			for k, v := range scope {
				stepScope[k] = v
			}

			if i := step.Inline; i != nil {
				for _, p := range i.Inputs.Parameters {
					stepScope["inputs.parameters."+p.Name] = placeholderGenerator.NextPlaceholder()
				}
			}

			err = resolveAllVariables(stepScope, ctx.globalParams, string(stepBytes), workflowTemplateValidation)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps %s", tmpl.Name, err.Error())
			}

			aggregate := len(step.WithItems) > 0 || step.WithParam != ""

			ctx.addOutputsToScope(resolvedTmpl, fmt.Sprintf("steps.%s", step.Name), scope, aggregate, false)

			// Validate the template again with actual arguments.
			_, err = ctx.validateTemplateHolder(&step, tmplCtx, &step.Arguments, workflowTemplateValidation)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
		}
	}
	return nil
}

func addItemsToScope(withItems []wfv1.Item, withParam string, withSequence *wfv1.Sequence, scope map[string]interface{}) error {
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
	scope[fmt.Sprintf("%s.id", prefix)] = true
	scope[fmt.Sprintf("%s.startedAt", prefix)] = true
	scope[fmt.Sprintf("%s.finishedAt", prefix)] = true
	scope[fmt.Sprintf("%s.hostNodeName", prefix)] = true
	if tmpl == nil {
		return
	}
	if tmpl.Daemon != nil && *tmpl.Daemon {
		scope[fmt.Sprintf("%s.ip", prefix)] = true
	}
	if tmpl.HasOutput() {
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
		case wfv1.TemplateTypeScript, wfv1.TemplateTypeContainerSet:
			scope[fmt.Sprintf("%s.outputs.result", prefix)] = true
			scope[fmt.Sprintf("%s.exitCode", prefix)] = true
			scope[fmt.Sprintf("%s.outputs.parameters", prefix)] = true
		default:
			scope[fmt.Sprintf("%s.outputs.parameters", prefix)] = true
		}
	}
	if isAncestor {
		scope[fmt.Sprintf("%s.status", prefix)] = true
	}
}

func validateOutputs(scope map[string]interface{}, globalParams map[string]string, tmpl *wfv1.Template, workflowTemplateValidation bool) error {
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
	err = resolveAllVariables(scope, globalParams, string(outputBytes), workflowTemplateValidation)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.outputs %s", tmpl.Name, err.Error())
	}

	for _, art := range tmpl.Outputs.Artifacts {
		artRef := fmt.Sprintf("outputs.artifacts.%s", art.Name)
		if tmpl.IsLeaf() {
			err = art.CleanPath()
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "error in templates.%s.%s: %s", tmpl.Name, artRef, err.Error())
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
			case wfv1.TemplateTypeContainer, wfv1.TemplateTypeContainerSet, wfv1.TemplateTypeScript:
				if param.ValueFrom.Path == "" {
					return errors.Errorf(errors.CodeBadRequest, "%s.path must be specified for %s templates", paramRef, tmplType)
				}
			case wfv1.TemplateTypeResource:
				if param.ValueFrom.JQFilter == "" && param.ValueFrom.JSONPath == "" {
					return errors.Errorf(errors.CodeBadRequest, "%s .jqFilter or jsonPath must be specified for %s templates", paramRef, tmplType)
				}
			case wfv1.TemplateTypeDAG, wfv1.TemplateTypeSteps:
				if param.ValueFrom.Parameter == "" && param.ValueFrom.Expression == "" {
					return errors.Errorf(errors.CodeBadRequest, "%s.parameter or expression must be specified for %s templates", paramRef, tmplType)
				}
				if param.ValueFrom.Expression != "" && param.ValueFrom.Parameter != "" {
					return errors.Errorf(errors.CodeBadRequest, "%s shouldn't have both `from` and `expression` specified in `ValueFrom` for %s templates", paramRef, tmplType)
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
	for _, value := range []string{param.ValueFrom.Path, param.ValueFrom.JQFilter, param.ValueFrom.JSONPath, param.ValueFrom.Parameter, param.ValueFrom.Expression} {
		if value != "" {
			paramTypes++
		}
	}
	if param.ValueFrom.Supplied != nil {
		paramTypes++
	}
	switch paramTypes {
	case 0:
		return errors.New(errors.CodeBadRequest, "valueFrom type unspecified. choose one of: path, jqFilter, jsonPath, parameter, raw, expression")
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
	dependencies map[string]map[string]common.DependencyType // map of DAG tasks, each one containing a map of [task it's dependent on] -> [dependency type]
}

func (d *dagValidationContext) GetTask(taskName string) *wfv1.DAGTask {
	task := d.tasks[taskName]
	return &task
}

func (d *dagValidationContext) GetTaskDependencies(taskName string) []string {
	dependencies := d.GetTaskDependenciesWithDependencyTypes(taskName)

	var dependencyTasks []string
	for task := range dependencies {
		dependencyTasks = append(dependencyTasks, task)
	}

	return dependencyTasks
}

func (d *dagValidationContext) GetTaskDependenciesWithDependencyTypes(taskName string) map[string]common.DependencyType {
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

func (ctx *templateValidationCtx) validateDAG(scope map[string]interface{}, tmplCtx *templateresolution.Context, tmpl *wfv1.Template, workflowTemplateValidation bool) error {
	err := validateNonLeaf(tmpl)
	if err != nil {
		return err
	}
	if len(tmpl.DAG.Tasks) == 0 {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s must have at least one task", tmpl.Name)
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
		dependencies: make(map[string]map[string]common.DependencyType),
	}
	err = sortDAGTasks(tmpl, dagValidationCtx)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s sorting failed: %s", tmpl.Name, err.Error())
	}

	err = validateWorkflowFieldNames(tmpl.DAG.Tasks)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks%s", tmpl.Name, err.Error())
	}

	resolvedTemplates := make(map[string]*wfv1.Template)

	// Verify dependencies for all tasks can be resolved as well as template names
	for _, task := range tmpl.DAG.Tasks {

		if (usingDepends || len(task.Dependencies) > 0) && '0' <= task.Name[0] && task.Name[0] <= '9' {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s name cannot begin with a digit when using either 'depends' or 'dependencies'", tmpl.Name, task.Name)
		}

		if usingDepends && len(task.Dependencies) > 0 {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s cannot use both 'depends' and 'dependencies' in the same DAG template", tmpl.Name)
		}

		if usingDepends && task.ContinueOn != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s cannot use 'continueOn' when using 'depends'. Instead use 'dep-task.Failed'/'dep-task.Errored'", tmpl.Name)
		}

		var args wfv1.ArgumentsProvider
		args = &FakeArguments{}
		if task.TemplateRef != nil {
			args = &task.Arguments
		}

		resolvedTmpl, err := ctx.validateTemplateHolder(&task, tmplCtx, args, workflowTemplateValidation)

		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}

		resolvedTemplates[task.Name] = resolvedTmpl

		prefix := fmt.Sprintf("tasks.%s", task.Name)
		aggregate := len(task.WithItems) > 0 || task.WithParam != ""
		ctx.addOutputsToScope(resolvedTmpl, prefix, scope, aggregate, false)

		err = common.ValidateTaskResults(&task)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}

		for depName, depType := range dagValidationCtx.GetTaskDependenciesWithDependencyTypes(task.Name) {
			task, ok := dagValidationCtx.tasks[depName]
			if !ok {
				return errors.Errorf(errors.CodeBadRequest,
					"templates.%s.tasks.%s dependency '%s' not defined",
					tmpl.Name, task.Name, depName)
			} else if depType == common.DependencyTypeItems && len(task.WithItems) == 0 && task.WithParam == "" && task.WithSequence == nil {
				return errors.Errorf(errors.CodeBadRequest,
					"templates.%s.tasks.%s dependency '%s' uses an items-based condition such as .AnySucceeded or .AllFailed but does not contain any items",
					tmpl.Name, task.Name, depName)
			}
		}
	}

	if err = verifyNoCycles(tmpl, dagValidationCtx); err != nil {
		return err
	}
	err = resolveAllVariables(scope, ctx.globalParams, tmpl.DAG.Target, workflowTemplateValidation)
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
		// add self status reference for  hooks
		if task.Hooks != nil {
			scope[fmt.Sprintf("%s.status", prefix)] = true
		}
		ctx.addOutputsToScope(resolvedTmpl, prefix, scope, false, false)
		if task.HasExitHook() {
			ctx.addOutputsToScope(resolvedTmpl, prefix, scope, false, false)
		}
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
		if i := task.Inline; i != nil {
			for _, p := range i.Inputs.Parameters {
				taskScope["inputs.parameters."+p.Name] = placeholderGenerator.NextPlaceholder()
			}
		}

		err = addItemsToScope(task.WithItems, task.WithParam, task.WithSequence, taskScope)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
		err = resolveAllVariables(taskScope, ctx.globalParams, string(taskBytes), workflowTemplateValidation)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
		err = validateArguments(fmt.Sprintf("templates.%s.tasks.%s.arguments.", tmpl.Name, task.Name), task.Arguments, false)
		if err != nil {
			return err
		}
		err = validateDAGTaskArgumentDependency(task.Arguments, ancestry)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
		// Validate the template again with actual arguments.
		_, err = ctx.validateTemplateHolder(&task, tmplCtx, &task.Arguments, workflowTemplateValidation)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
	}

	return nil
}

func validateDAGTaskArgumentDependency(arguments wfv1.Arguments, ancestry []string) error {
	ancestryMap := make(map[string]struct{}, len(ancestry))
	for _, a := range ancestry {
		ancestryMap[a] = struct{}{}
	}

	for _, param := range arguments.Parameters {
		if param.Value != nil && strings.HasPrefix(param.Value.String(), "{{tasks.") {
			// All parameter values should have been validated, so
			// index 1 should exist.
			refTaskName := strings.Split(param.Value.String(), ".")[1]

			if _, dependencyExists := ancestryMap[refTaskName]; !dependencyExists {
				return errors.Errorf(errors.CodeBadRequest, "missing dependency '%s' for parameter '%s'", refTaskName, param.Name)
			}
		}
	}

	for _, artifact := range arguments.Artifacts {
		if strings.HasPrefix(artifact.From, "{{tasks.") {
			// All parameter values should have been validated, so
			// index 1 should exist.
			refTaskName := strings.Split(artifact.From, ".")[1]

			if _, dependencyExists := ancestryMap[refTaskName]; !dependencyExists {
				return errors.Errorf(errors.CodeBadRequest, "missing dependency '%s' for artifact '%s'", refTaskName, artifact.Name)
			}
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

func sortDAGTasks(tmpl *wfv1.Template, ctx *dagValidationContext) error {
	taskMap := make(map[string]*wfv1.DAGTask, len(tmpl.DAG.Tasks))
	sortingGraph := make([]*sorting.TopologicalSortingNode, len(tmpl.DAG.Tasks))
	for index := range tmpl.DAG.Tasks {
		task := tmpl.DAG.Tasks[index]
		taskMap[task.Name] = &task
		dependenciesMap, _ := common.GetTaskDependencies(&task, ctx)
		var dependencies []string
		for taskName := range dependenciesMap {
			dependencies = append(dependencies, taskName)
		}
		sortingGraph[index] = &sorting.TopologicalSortingNode{
			NodeName:     task.Name,
			Dependencies: dependencies,
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
	return tmpl.Name
}
