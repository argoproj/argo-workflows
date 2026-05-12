package validate

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apivalidation "k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sorting"
	"github.com/argoproj/argo-workflows/v4/util/template"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/artifacts/hdfs"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

// Opts provides options when linting
type Opts struct {
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
	Opts

	// globalParams keeps track of variables which are available the global
	// scope and can be referenced from anywhere.
	globalParams map[string]string
	// results tracks if validation has already been run on a template
	results map[string]bool
	// wf is the Workflow resource which is used to validate templates.
	// It will be omitted in WorkflowTemplate validation.
	wf *wfv1.Workflow
}

func newTemplateValidationCtx(wf *wfv1.Workflow, opts Opts) *templateValidationCtx {
	globalParams := make(map[string]string)
	globalParams[varkeys.WorkflowName.Template()] = placeholderGenerator.NextPlaceholder()
	globalParams[varkeys.WorkflowNamespace.Template()] = placeholderGenerator.NextPlaceholder()
	globalParams[varkeys.WorkflowMainEntrypoint.Template()] = placeholderGenerator.NextPlaceholder()
	globalParams[varkeys.WorkflowServiceAccountName.Template()] = placeholderGenerator.NextPlaceholder()
	globalParams[varkeys.WorkflowUID.Template()] = placeholderGenerator.NextPlaceholder()
	return &templateValidationCtx{
		Opts:         opts,
		globalParams: globalParams,
		results:      make(map[string]bool),
		wf:           wf,
	}
}

// Magic placeholder values written to validation scopes when a name can only
// be resolved at runtime (e.g. an output globalName that is itself a
// parameter). resolveAllVariables() looks for these to decide whether to
// accept any reference under the corresponding prefix. Derived from the
// varkeys templates so the catalog stays the single source of truth.
var (
	// anyItemMagicValue is a magic value set in addItemsToScope() and checked in
	// resolveAllVariables() to determine if any {{item.name}} can be accepted during
	// variable resolution (to support withParam)
	anyItemMagicValue                    = varkeys.ItemByKey.Concretize("*")
	anyWorkflowOutputParameterMagicValue = varkeys.WorkflowOutputsParameterByName.Concretize("*")
	anyWorkflowOutputArtifactMagicValue  = varkeys.WorkflowOutputsArtifactByName.Concretize("*")
)

const (
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

var resourceManifestExpressionPattern = regexp.MustCompile(`{{\s*=\s*(.+?)\s*}}`)

func SubstituteResourceManifestExpressions(manifest string) string {
	var substitutions = make(map[string]string)
	for _, match := range resourceManifestExpressionPattern.FindAllStringSubmatch(manifest, -1) {
		substitutions[match[1]] = placeholderGenerator.NextPlaceholder()
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

// Workflow accepts a workflow and performs validation against it.
func Workflow(ctx context.Context, wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, wf *wfv1.Workflow, wfDefaults *wfv1.Workflow, opts Opts) error {
	tctx := newTemplateValidationCtx(wf, opts)

	tmplCtx := templateresolution.NewContext(wftmplGetter, cwftmplGetter, wf, wf, logging.RequireLoggerFromContext(ctx))
	var wfSpecHolder wfv1.WorkflowSpecHolder
	var wfTmplRef *wfv1.TemplateRef
	var err error

	if len(wf.Name) > maxCharsInObjectName {
		return fmt.Errorf("workflow name %q must not be more than 63 characters long (currently %d)", wf.Name, len(wf.Name))
	}

	entrypoint := wf.Spec.Entrypoint

	hasWorkflowTemplateRef := wf.Spec.WorkflowTemplateRef != nil

	if hasWorkflowTemplateRef {
		refErr := WorkflowTemplateRefFields(wf.Spec)
		if refErr != nil {
			return refErr
		}
		if wf.Spec.WorkflowTemplateRef.ClusterScope {
			wfSpecHolder, err = cwftmplGetter.Get(ctx, wf.Spec.WorkflowTemplateRef.Name)
		} else {
			wfSpecHolder, err = wftmplGetter.Get(ctx, wf.Spec.WorkflowTemplateRef.Name)
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
	allowEmptyValues := tctx.Lint || (tctx.WorkflowTemplateValidation && !tctx.Submit)
	err = validateArguments("spec.arguments.", wfArgs, allowEmptyValues)
	if err != nil {
		return err
	}
	if len(wfArgs.Parameters) > 0 {
		tctx.globalParams[varkeys.WorkflowParametersAll.Template()] = placeholderGenerator.NextPlaceholder()
		tctx.globalParams[varkeys.WorkflowParametersJSON.Template()] = placeholderGenerator.NextPlaceholder()
	}

	for _, param := range wfArgs.Parameters {
		if param.Name != "" {
			if param.Value != nil {
				tctx.globalParams[varkeys.WorkflowParametersByName.Concretize(param.Name)] = param.Value.String()
			} else {
				tctx.globalParams[varkeys.WorkflowParametersByName.Concretize(param.Name)] = placeholderGenerator.NextPlaceholder()
			}
		}
	}

	annotationSources := [][]string{slices.Collect(maps.Keys(wf.Annotations))}
	labelSources := [][]string{slices.Collect(maps.Keys(wf.Labels))}
	if wf.Spec.WorkflowMetadata != nil {
		annotationSources = append(annotationSources, slices.Collect(maps.Keys(wf.Spec.WorkflowMetadata.Annotations)))
		labelSources = append(labelSources, slices.Collect(maps.Keys(wf.Spec.WorkflowMetadata.Labels)), slices.Collect(maps.Keys(wf.Spec.WorkflowMetadata.LabelsFrom)))
	}
	if wfDefaults != nil && wfDefaults.Spec.WorkflowMetadata != nil {
		annotationSources = append(annotationSources, slices.Collect(maps.Keys(wfDefaults.Spec.WorkflowMetadata.Annotations)))
		labelSources = append(labelSources, slices.Collect(maps.Keys(wfDefaults.Spec.WorkflowMetadata.Labels)), slices.Collect(maps.Keys(wfDefaults.Spec.WorkflowMetadata.LabelsFrom)))
	}
	if wf.Spec.WorkflowTemplateRef != nil && wfSpecHolder.GetWorkflowSpec().WorkflowMetadata != nil {
		annotationSources = append(annotationSources, slices.Collect(maps.Keys(wfSpecHolder.GetWorkflowSpec().WorkflowMetadata.Annotations)))
		labelSources = append(labelSources, slices.Collect(maps.Keys(wfSpecHolder.GetWorkflowSpec().WorkflowMetadata.Labels)), slices.Collect(maps.Keys(wfSpecHolder.GetWorkflowSpec().WorkflowMetadata.LabelsFrom)))
	}
	mergedAnnotations := getUniqueKeys(annotationSources...)
	mergedLabels := getUniqueKeys(labelSources...)

	for k := range mergedAnnotations {
		tctx.globalParams[varkeys.WorkflowAnnotationsByName.Concretize(k)] = placeholderGenerator.NextPlaceholder()
	}
	tctx.globalParams[varkeys.WorkflowAnnotationsAll.Template()] = placeholderGenerator.NextPlaceholder()
	tctx.globalParams[varkeys.WorkflowAnnotationsJSON.Template()] = placeholderGenerator.NextPlaceholder()

	for k := range mergedLabels {
		tctx.globalParams[varkeys.WorkflowLabelsByName.Concretize(k)] = placeholderGenerator.NextPlaceholder()
	}
	tctx.globalParams[varkeys.WorkflowLabelsAll.Template()] = placeholderGenerator.NextPlaceholder()
	tctx.globalParams[varkeys.WorkflowLabelsJSON.Template()] = placeholderGenerator.NextPlaceholder()

	if wf.Spec.Priority != nil {
		tctx.globalParams[varkeys.WorkflowPriority.Template()] = strconv.Itoa(int(*wf.Spec.Priority))
	}
	tctx.globalParams[varkeys.WorkflowStatus.Template()] = placeholderGenerator.NextPlaceholder()

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
		_, err = tctx.validateTemplateHolder(ctx, tmpl, tmplCtx, args, opts.WorkflowTemplateValidation)
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
		tctx.globalParams[varkeys.WorkflowFailures.Template()] = placeholderGenerator.NextPlaceholder()

		// Check if any template has parametrized global artifacts, if so enable global artifact resolution for exit handlers
		hasParametrizedGlobalArtifacts := false
		for _, tmpl := range wf.Spec.Templates {
			for _, art := range tmpl.Outputs.Artifacts {
				if art.GlobalName != "" && isParameter(art.GlobalName) {
					hasParametrizedGlobalArtifacts = true
					break
				}
			}
			if hasParametrizedGlobalArtifacts {
				break
			}
		}
		if hasWorkflowTemplateRef && !hasParametrizedGlobalArtifacts {
			// Also check the referenced workflow template
			for _, tmpl := range wfSpecHolder.GetWorkflowSpec().Templates {
				for _, art := range tmpl.Outputs.Artifacts {
					if art.GlobalName != "" && isParameter(art.GlobalName) {
						hasParametrizedGlobalArtifacts = true
						break
					}
				}
				if hasParametrizedGlobalArtifacts {
					break
				}
			}
		}
		if hasParametrizedGlobalArtifacts {
			tctx.globalParams[anyWorkflowOutputArtifactMagicValue] = "true"
		}

		_, err = tctx.validateTemplateHolder(ctx, tmplHolder, tmplCtx, &wf.Spec.Arguments, opts.WorkflowTemplateValidation)
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
			_, err := tctx.validateTemplateHolder(ctx, &wfv1.WorkflowStep{TemplateRef: wf.Spec.WorkflowTemplateRef.ToTemplateRef(template.Name)}, tmplCtx, &FakeArguments{}, opts.WorkflowTemplateValidation)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s %s", template.Name, err.Error())
			}
		}
		return nil
	}
	// If the templates are inlined in Workflow, then the inlined templates will be validated.
	for _, template := range wf.Spec.Templates {
		_, err := tctx.validateTemplateHolder(ctx, &wfv1.WorkflowStep{Template: template.Name}, tmplCtx, &FakeArguments{}, opts.WorkflowTemplateValidation)
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

func WorkflowTemplateRefFields(wfSpec wfv1.WorkflowSpec) error {
	if len(wfSpec.Templates) > 0 {
		return errors.Errorf(errors.CodeBadRequest, "Templates is invalid field in spec if workflow referred WorkflowTemplate reference")
	}
	return nil
}

// WorkflowTemplate accepts a workflow template and performs validation against it.
func WorkflowTemplate(ctx context.Context, wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, wftmpl *wfv1.WorkflowTemplate, wfDefaults *wfv1.Workflow, opts Opts) error {
	if len(wftmpl.Name) > maxCharsInObjectName {
		return fmt.Errorf("workflow template name %q must not be more than 63 characters long (currently %d)", wftmpl.Name, len(wftmpl.Name))
	}

	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Labels:      wftmpl.Labels,
			Annotations: wftmpl.Annotations,
		},
		Spec: wftmpl.Spec,
	}
	opts.IgnoreEntrypoint = wf.Spec.Entrypoint == ""
	opts.WorkflowTemplateValidation = true
	return Workflow(ctx, wftmplGetter, cwftmplGetter, wf, wfDefaults, opts)
}

// ClusterWorkflowTemplate accepts a cluster workflow template and performs validation against it.
func ClusterWorkflowTemplate(ctx context.Context, wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, cwftmpl *wfv1.ClusterWorkflowTemplate, wfDefaults *wfv1.Workflow, opts Opts) error {
	if len(cwftmpl.Name) > maxCharsInObjectName {
		return fmt.Errorf("cluster workflow template name %q must not be more than 63 characters long (currently %d)", cwftmpl.Name, len(cwftmpl.Name))
	}

	wf := &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Labels:      cwftmpl.Labels,
			Annotations: cwftmpl.Annotations,
		},
		Spec: cwftmpl.Spec,
	}
	opts.IgnoreEntrypoint = wf.Spec.Entrypoint == ""
	opts.WorkflowTemplateValidation = true
	return Workflow(ctx, wftmplGetter, cwftmplGetter, wf, wfDefaults, opts)
}

// CronWorkflow validates a CronWorkflow
func CronWorkflow(ctx context.Context, wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, cronWf *wfv1.CronWorkflow, wfDefaults *wfv1.Workflow) error {
	// CronWorkflows have fewer max chars allowed in their name because when workflows are created from them, they
	// are appended with the unix timestamp (`-1615836720`). This lower character allowance allows for that timestamp
	// to still fit within the 63 character maximum.
	if len(cronWf.Name) > maxCharsInCronWorkflowName {
		return fmt.Errorf("cron workflow name %q must not be more than 52 characters long (currently %d)", cronWf.Name, len(cronWf.Name))
	}

	if len(cronWf.Spec.Schedules) == 0 {
		return fmt.Errorf("cron workflow must have at least one schedule")
	}

	for _, schedule := range cronWf.Spec.GetSchedules() {
		if _, err := cron.ParseStandard(schedule); err != nil {
			return errors.Errorf(errors.CodeBadRequest, "cron schedule %s is malformed: %s", schedule, err)
		}
	}

	if cronWf.Spec.Timezone != "" {
		if _, err := time.LoadLocation(cronWf.Spec.Timezone); err != nil {
			return errors.Errorf(errors.CodeBadRequest, "invalid timezone %q: %s", cronWf.Spec.Timezone, err)
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

	err := Workflow(ctx, wftmplGetter, cwftmplGetter, wf, wfDefaults, Opts{})
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "cannot validate Workflow: %s", err)
	}
	return nil
}

func (tctx *templateValidationCtx) validateInitContainers(containers []wfv1.UserContainer) error {
	for _, container := range containers {
		if len(container.Name) == 0 {
			return errors.Errorf(errors.CodeBadRequest, "initContainers must all have container name")
		}
	}
	return nil
}

func (tctx *templateValidationCtx) validateTemplate(ctx context.Context, tmpl *wfv1.Template, tmplCtx *templateresolution.TemplateContext, args wfv1.ArgumentsProvider, workflowTemplateValidation bool) error {
	if err := validateTemplateType(tmpl); err != nil {
		return err
	}

	scope, err := validateInputs(tmpl)
	if err != nil {
		return err
	}

	if initErr := tctx.validateInitContainers(tmpl.InitContainers); initErr != nil {
		return initErr
	}

	localParams := make(map[string]string)
	if tmpl.IsPodType() {
		localParams[varkeys.PodName.Template()] = placeholderGenerator.NextPlaceholder()
		scope[varkeys.PodName.Template()] = placeholderGenerator.NextPlaceholder()
	}
	if tmpl.RetryStrategy != nil {
		localParams[varkeys.Retries.Template()] = placeholderGenerator.NextPlaceholder()
		localParams[varkeys.RetriesLastExitCode.Template()] = placeholderGenerator.NextPlaceholder()
		localParams[varkeys.RetriesLastStatus.Template()] = placeholderGenerator.NextPlaceholder()
		localParams[varkeys.RetriesLastDuration.Template()] = placeholderGenerator.NextPlaceholder()
		localParams[varkeys.RetriesLastMessage.Template()] = placeholderGenerator.NextPlaceholder()
		scope[varkeys.Retries.Template()] = placeholderGenerator.NextPlaceholder()
		scope[varkeys.RetriesLastExitCode.Template()] = placeholderGenerator.NextPlaceholder()
		scope[varkeys.RetriesLastStatus.Template()] = placeholderGenerator.NextPlaceholder()
		scope[varkeys.RetriesLastDuration.Template()] = placeholderGenerator.NextPlaceholder()
		scope[varkeys.RetriesLastMessage.Template()] = placeholderGenerator.NextPlaceholder()
	}
	if tmpl.IsLeaf() {
		for _, art := range tmpl.Outputs.Artifacts {
			if art.Path != "" {
				scope[varkeys.OutputsArtifactPathByName.Concretize(art.Name)] = true
			}
		}
		for _, param := range tmpl.Outputs.Parameters {
			if param.ValueFrom != nil && param.ValueFrom.Path != "" {
				scope[varkeys.OutputsParameterPathByName.Concretize(param.Name)] = true
			}
		}
	}

	newTmpl, err := common.ProcessArgs(ctx, tmpl, args, tctx.globalParams, localParams, true, "", nil)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s %s", tmpl.Name, err)
	}

	if newTmpl.Timeout != "" {
		if !newTmpl.IsLeaf() {
			return fmt.Errorf("%s template doesn't support timeout field", newTmpl.GetType())
		}
		// Check timeout should not be a whole number
		_, atoiErr := strconv.Atoi(newTmpl.Timeout)
		if atoiErr == nil {
			return fmt.Errorf("%s has invalid duration format in timeout", newTmpl.Name)
		}
	}

	templateScope := tmplCtx.GetTemplateScope()
	tmplID := getTemplateID(tmpl)
	_, ok := tctx.results[templateScope+tmplID]
	if ok {
		// we can skip the rest since it has been validated.
		return nil
	}
	tctx.results[templateScope+tmplID] = true

	for globalVar, val := range tctx.globalParams {
		scope[globalVar] = val
	}
	switch newTmpl.GetType() {
	case wfv1.TemplateTypeSteps:
		err = tctx.validateSteps(ctx, scope, tmplCtx, newTmpl, workflowTemplateValidation)
	case wfv1.TemplateTypeDAG:
		err = tctx.validateDAG(ctx, scope, tmplCtx, newTmpl, workflowTemplateValidation)
	default:
		err = tctx.validateLeaf(scope, tmplCtx, newTmpl, workflowTemplateValidation)
	}
	if err != nil {
		return err
	}
	err = validateOutputs(scope, tctx.globalParams, newTmpl, workflowTemplateValidation)
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
func VerifyResolvedVariables(obj any) error {
	str, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return template.Validate(string(str), func(tag string) error {
		return errors.Errorf(errors.CodeBadRequest, "failed to resolve {{%s}}", tag)
	})
}

// validateTemplateHolder validates a template holder and returns the validated template.
func (tctx *templateValidationCtx) validateTemplateHolder(ctx context.Context, tmplHolder wfv1.TemplateReferenceHolder, tmplCtx *templateresolution.TemplateContext, args wfv1.ArgumentsProvider, workflowTemplateValidation bool) (*wfv1.Template, error) {
	tmplRef := tmplHolder.GetTemplateRef()
	tmplName := tmplHolder.GetTemplateName()
	if tmplRef != nil {
		if tmplName != "" {
			return nil, errors.New(errors.CodeBadRequest, "template name cannot be specified with templateRef")
		}
		if tmplRef.Name == "" {
			return nil, errors.New(errors.CodeBadRequest, "resource name is required")
		}
		if tmplRef.Template == "" {
			return nil, errors.New(errors.CodeBadRequest, "template name is required")
		}
		if err := VerifyResolvedVariables(tmplRef); err != nil {
			logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "template reference needs resolution")
			return nil, nil
		}
	} else if tmplName != "" {
		_, err := tmplCtx.GetTemplateByName(ctx, tmplName)
		if err != nil {
			var argoerr errors.ArgoError
			if stderrors.As(err, &argoerr) && argoerr.Code() == errors.CodeNotFound {
				return nil, errors.Errorf(errors.CodeBadRequest, "template name '%s' undefined", tmplName)
			}
			return nil, err
		}
	}

	tmplCtx, resolvedTmpl, _, err := tmplCtx.ResolveTemplate(ctx, tmplHolder)
	if err != nil {
		var argoerr errors.ArgoError
		if stderrors.As(err, &argoerr) && argoerr.Code() == errors.CodeNotFound {
			if tmplRef != nil && strings.Contains(tmplRef.Template, template.PlaceholderPrefix) {
				// internal placeholder indicates this is a dynamic template, skip validation
				return nil, nil
			}
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

	return resolvedTmpl, tctx.validateTemplate(ctx, resolvedTmpl, tmplCtx, args, workflowTemplateValidation)
}

// validateTemplateType validates that only one template type is defined
func validateTemplateType(tmpl *wfv1.Template) error {
	numTypes := 0
	for _, tmplType := range []any{tmpl.Container, tmpl.ContainerSet, tmpl.Steps, tmpl.Script, tmpl.Resource, tmpl.DAG, tmpl.Suspend, tmpl.Data, tmpl.HTTP, tmpl.Plugin} {
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

func validateInputs(tmpl *wfv1.Template) (map[string]any, error) {
	err := validateWorkflowFieldNames(tmpl.Inputs.Parameters)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.inputs.parameters%s", tmpl.Name, err.Error())
	}
	err = validateWorkflowFieldNames(tmpl.Inputs.Artifacts)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.inputs.artifacts%s", tmpl.Name, err.Error())
	}
	scope := make(map[string]any)
	for _, param := range tmpl.Inputs.Parameters {
		scope[varkeys.InputsParameterByName.Concretize(param.Name)] = true
	}
	if len(tmpl.Inputs.Parameters) > 0 {
		scope[varkeys.InputsParametersAll.Template()] = true
	}

	for _, art := range tmpl.Inputs.Artifacts {
		artRef := varkeys.InputsArtifactByName.Concretize(art.Name)
		scope[artRef] = true
		if tmpl.IsLeaf() {
			err = art.CleanPath()
			if err != nil {
				return nil, errors.Errorf(errors.CodeBadRequest, "error in templates.%s.%s: %s", tmpl.Name, artRef, err.Error())
			}
			scope[varkeys.InputsArtifactPathByName.Concretize(art.Name)] = true
		} else if art.Path != "" {
			return nil, errors.Errorf(errors.CodeBadRequest, "templates.%s.%s.path only valid in container/script templates", tmpl.Name, artRef)
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
func resolveAllVariables(scope map[string]any, globalParams map[string]string, tmplStr string, workflowTemplateValidation bool) error {
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
			switch {
			case (trimmedTag == varkeys.Item.Template() || strings.HasPrefix(trimmedTag, varkeys.ItemByKey.Concretize(""))) && allowAllItemRefs:
				// we are *probably* referencing a undetermined item using withParam
				// NOTE: this is far from foolproof.
			case strings.HasPrefix(trimmedTag, varkeys.WorkflowOutputsParameterByName.Concretize("")) && allowAllWorkflowOutputParameterRefs:
				// Allow runtime resolution of workflow output parameter names
			case strings.HasPrefix(trimmedTag, varkeys.WorkflowOutputsArtifactByName.Concretize("")) && allowAllWorkflowOutputArtifactRefs:
				// Allow runtime resolution of workflow output artifact names
			case strings.HasPrefix(trimmedTag, "outputs."):
				// We are self referencing for metric emission, allow it.
			case strings.HasPrefix(trimmedTag, varkeys.WorkflowCreationTimestamp.Template()):
			case strings.HasPrefix(trimmedTag, varkeys.WorkflowScheduledTime.Template()):
				// Allow runtime resolution for "scheduledTime" which will pass from CronWorkflow
			case strings.HasPrefix(trimmedTag, varkeys.WorkflowDuration.Template()):
			case strings.HasPrefix(trimmedTag, varkeys.TasksName.Template()):
			case strings.HasPrefix(trimmedTag, varkeys.StepsName.Template()):
			case strings.HasPrefix(trimmedTag, varkeys.NodeName.Template()):
			case strings.HasPrefix(trimmedTag, varkeys.WorkflowParametersAll.Template()) && workflowTemplateValidation:
				// If we are simply validating a WorkflowTemplate in isolation, some of the parameters may come from the Workflow that uses it
			default:
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

func (tctx *templateValidationCtx) validateLeaf(scope map[string]any, tmplCtx *templateresolution.TemplateContext, tmpl *wfv1.Template, workflowTemplateValidation bool) error {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	err = resolveAllVariables(scope, tctx.globalParams, string(tmplBytes), workflowTemplateValidation)
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
			mountPaths[art.Path] = varkeys.InputsArtifactByName.Concretize(art.Name)
		}
		if tmpl.Container.Image == "" {
			switch baseTemplate := tmplCtx.GetCurrentTemplateBase().(type) {
			case *wfv1.Workflow:
				if baseTemplate.Spec.TemplateDefaults == nil || baseTemplate.Spec.TemplateDefaults.Container == nil || baseTemplate.Spec.TemplateDefaults.Container.Image == "" {
					return errors.Errorf(errors.CodeBadRequest, "templates.%s.container.image may not be empty", tmpl.Name)
				}
			case *wfv1.WorkflowTemplate:
				if baseTemplate.Spec.TemplateDefaults == nil || baseTemplate.Spec.TemplateDefaults.Container == nil || baseTemplate.Spec.TemplateDefaults.Container.Image == "" {
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
				var obj any

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
				if baseTemplate.Spec.TemplateDefaults == nil || baseTemplate.Spec.TemplateDefaults.Script == nil || baseTemplate.Spec.TemplateDefaults.Script.Image == "" {
					return errors.Errorf(errors.CodeBadRequest, "templates.%s.script.image may not be empty", tmpl.Name)
				}
			case *wfv1.WorkflowTemplate:
				if baseTemplate.Spec.TemplateDefaults == nil || baseTemplate.Spec.TemplateDefaults.Script == nil || baseTemplate.Spec.TemplateDefaults.Script.Image == "" {
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
	fieldToSlices := map[string]any{
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
			if !slices.Contains(param.Enum, *param.Value) {
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

func (tctx *templateValidationCtx) validateSteps(ctx context.Context, scope map[string]any, tmplCtx *templateresolution.TemplateContext, tmpl *wfv1.Template, workflowTemplateValidation bool) error {
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
			scope[varkeys.StepsNodeRef.Status.Concretize(step.Name)] = true
			err := addItemsToScope(step.WithItems, step.WithParam, step.WithSequence, scope)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
			err = validateArguments(fmt.Sprintf("templates.%s.steps[%d].%s.arguments.", tmpl.Name, i, step.Name), step.Arguments, false)
			if err != nil {
				return err
			}
			resolvedTmpl, err := tctx.validateTemplateHolder(ctx, &step, tmplCtx, &FakeArguments{}, workflowTemplateValidation)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}

			if step.HasExitHook() {
				tctx.addOutputsToScope(ctx, resolvedTmpl, varkeys.StepsNodeRef, varkeys.StepsAggregate, step.Name, scope, false, false)
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

			stepScope := make(map[string]any)
			maps.Copy(stepScope, scope)

			if i := step.Inline; i != nil {
				for _, p := range i.Inputs.Parameters {
					stepScope[varkeys.InputsParameterByName.Concretize(p.Name)] = placeholderGenerator.NextPlaceholder()
				}
			}

			err = resolveAllVariables(stepScope, tctx.globalParams, string(stepBytes), workflowTemplateValidation)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps %s", tmpl.Name, err.Error())
			}

			aggregate := len(step.WithItems) > 0 || step.WithParam != ""

			tctx.addOutputsToScope(ctx, resolvedTmpl, varkeys.StepsNodeRef, varkeys.StepsAggregate, step.Name, scope, aggregate, false)

			// Validate the template again with actual arguments.
			_, err = tctx.validateTemplateHolder(ctx, &step, tmplCtx, &step.Arguments, workflowTemplateValidation)
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "templates.%s.steps[%d].%s %s", tmpl.Name, i, step.Name, err.Error())
			}
		}
	}
	return nil
}

func addItemsToScope(withItems []wfv1.Item, withParam string, withSequence *wfv1.Sequence, scope map[string]any) error {
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
	switch {
	case len(withItems) > 0:
		for i := range withItems {
			val := withItems[i]
			switch val.GetType() {
			case wfv1.String, wfv1.Number, wfv1.Bool:
				scope[varkeys.Item.Template()] = true
			case wfv1.List:
				for i := range val.GetListVal() {
					scope[varkeys.ItemByKey.Concretize(fmt.Sprintf("[%v]", i))] = true
				}
			case wfv1.Map:
				for itemKey := range val.GetMapVal() {
					scope[varkeys.ItemByKey.Concretize(itemKey)] = true
				}
			default:
				return fmt.Errorf("unsupported withItems type: %v", val)
			}
		}
	case withParam != "":
		scope[varkeys.Item.Template()] = true
		// 'item.*' is magic placeholder value which resolveAllVariables() will look for
		// when considering if all variables are resolveable.
		scope[anyItemMagicValue] = true
	case withSequence != nil:
		if withSequence.Count != nil && withSequence.End != nil {
			return errors.New(errors.CodeBadRequest, "only one of count or end can be defined in withSequence")
		}
		scope[varkeys.Item.Template()] = true
	}
	return nil
}

func (tctx *templateValidationCtx) addOutputsToScope(ctx context.Context, tmpl *wfv1.Template, ref varkeys.NodeRefKeys, agg varkeys.AggregateKeys, name string, scope map[string]any, aggregate bool, isAncestor bool) {
	scope[ref.ID.Concretize(name)] = true
	scope[ref.StartedAt.Concretize(name)] = true
	scope[ref.FinishedAt.Concretize(name)] = true
	scope[ref.HostNodeName.Concretize(name)] = true
	if tmpl == nil {
		return
	}
	if tmpl.Daemon != nil && *tmpl.Daemon {
		scope[ref.IP.Concretize(name)] = true
	}
	if tmpl.HasOutput() {
		scope[ref.OutputsResult.Concretize(name)] = true
		scope[ref.ExitCode.Concretize(name)] = true
	}
	for _, param := range tmpl.Outputs.Parameters {
		scope[ref.OutputsParameterByName.Concretize(name, param.Name)] = true
		if param.GlobalName != "" {
			if !isParameter(param.GlobalName) {
				globalParamName := varkeys.WorkflowOutputsParameterByName.Concretize(param.GlobalName)
				scope[globalParamName] = true
				tctx.globalParams[globalParamName] = placeholderGenerator.NextPlaceholder()
			} else {
				logging.RequireLoggerFromContext(ctx).WithField("globalName", param.GlobalName).Warn(ctx, "GlobalName is a parameter and won't be validated until runtime")
				scope[anyWorkflowOutputParameterMagicValue] = true
			}
		}
	}
	for _, art := range tmpl.Outputs.Artifacts {
		scope[ref.OutputsArtifactByName.Concretize(name, art.Name)] = true
		if art.GlobalName != "" {
			if !isParameter(art.GlobalName) {
				globalArtName := varkeys.WorkflowOutputsArtifactByName.Concretize(art.GlobalName)
				scope[globalArtName] = true
				tctx.globalParams[globalArtName] = placeholderGenerator.NextPlaceholder()
			} else {
				logging.RequireLoggerFromContext(ctx).WithField("globalName", art.GlobalName).Warn(ctx, "GlobalName is a parameter and won't be validated until runtime")
				scope[anyWorkflowOutputArtifactMagicValue] = true
			}
		}
	}
	if aggregate {
		switch tmpl.GetType() {
		// Not that we don't also include TemplateTypeContainer here, even though it uses `outputs.result` it uses
		// `outputs.parameters` as its aggregator.
		case wfv1.TemplateTypeScript, wfv1.TemplateTypeContainerSet:
			scope[ref.OutputsResult.Concretize(name)] = true
			scope[ref.ExitCode.Concretize(name)] = true
			scope[agg.Parameters.Concretize(name)] = true
		default:
			scope[agg.Parameters.Concretize(name)] = true
		}
	}
	if isAncestor {
		scope[ref.Status.Concretize(name)] = true
	}
}

func validateOutputs(scope map[string]any, globalParams map[string]string, tmpl *wfv1.Template, workflowTemplateValidation bool) error {
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
		} else if art.Path != "" {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.%s.path only valid in container/script templates", tmpl.Name, artRef)
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
		return errors.Errorf(errors.CodeBadRequest, "%s has both valueFrom and value specified, choose one", paramRef)
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
func validateWorkflowFieldNames(slice any) error {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return errors.InternalErrorf("validateWorkflowFieldNames given a non-slice type")
	}
	items := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		items[i] = s.Index(i).Interface()
	}
	names := make(map[string]bool)
	getNameFieldValue := func(val any) (string, error) {
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
		if t == reflect.TypeFor[wfv1.Parameter]() || t == reflect.TypeFor[wfv1.Artifact]() {
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

func (d *dagValidationContext) GetTask(ctx context.Context, taskName string) *wfv1.DAGTask {
	task := d.tasks[taskName]
	return &task
}

func (d *dagValidationContext) GetTaskDependencies(ctx context.Context, taskName string) []string {
	dependencies := d.GetTaskDependenciesWithDependencyTypes(ctx, taskName)

	var dependencyTasks []string
	for task := range dependencies {
		dependencyTasks = append(dependencyTasks, task)
	}

	return dependencyTasks
}

func (d *dagValidationContext) GetTaskDependenciesWithDependencyTypes(ctx context.Context, taskName string) map[string]common.DependencyType {
	if dependencies, ok := d.dependencies[taskName]; ok {
		return dependencies
	}
	task := d.GetTask(ctx, taskName)
	dependencies, _ := common.GetTaskDependencies(ctx, task, d)
	d.dependencies[taskName] = dependencies
	return d.dependencies[taskName]
}

func (d *dagValidationContext) GetTaskFinishedAtTime(ctx context.Context, taskName string) time.Time {
	return time.Now()
}

func (tctx *templateValidationCtx) validateDAG(ctx context.Context, scope map[string]any, tmplCtx *templateresolution.TemplateContext, tmpl *wfv1.Template, workflowTemplateValidation bool) error {
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
	err = sortDAGTasks(ctx, tmpl, dagValidationCtx)
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

		resolvedTmpl, validateErr := tctx.validateTemplateHolder(ctx, &task, tmplCtx, &FakeArguments{}, workflowTemplateValidation)

		if validateErr != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, validateErr.Error())
		}

		resolvedTemplates[task.Name] = resolvedTmpl

		aggregate := len(task.WithItems) > 0 || task.WithParam != ""
		tctx.addOutputsToScope(ctx, resolvedTmpl, varkeys.TasksNodeRef, varkeys.TasksAggregate, task.Name, scope, aggregate, false)

		err = common.ValidateTaskResults(&task)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}

		for depName, depType := range dagValidationCtx.GetTaskDependenciesWithDependencyTypes(ctx, task.Name) {
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

	if err = verifyNoCycles(ctx, tmpl, dagValidationCtx); err != nil {
		return err
	}
	err = resolveAllVariables(scope, tctx.globalParams, tmpl.DAG.Target, workflowTemplateValidation)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "templates.%s.targets %s", tmpl.Name, err.Error())
	}
	if err = validateDAGTargets(tmpl, dagValidationCtx.tasks); err != nil {
		return err
	}

	for _, task := range tmpl.DAG.Tasks {
		resolvedTmpl := resolvedTemplates[task.Name]
		// add all tasks outputs to scope so that a nested DAGs can have outputs
		// add self status reference for  hooks
		if task.Hooks != nil {
			scope[varkeys.TasksNodeRef.Status.Concretize(task.Name)] = true
		}
		tctx.addOutputsToScope(ctx, resolvedTmpl, varkeys.TasksNodeRef, varkeys.TasksAggregate, task.Name, scope, false, false)
		if task.HasExitHook() {
			tctx.addOutputsToScope(ctx, resolvedTmpl, varkeys.TasksNodeRef, varkeys.TasksAggregate, task.Name, scope, false, false)
		}
		taskBytes, err := json.Marshal(task)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		taskScope := make(map[string]any)
		maps.Copy(taskScope, scope)
		ancestry := common.GetTaskAncestry(ctx, dagValidationCtx, task.Name)
		for _, ancestor := range ancestry {
			ancestorTask := dagValidationCtx.GetTask(ctx, ancestor)
			resolvedTmpl := resolvedTemplates[ancestor]
			aggregate := len(ancestorTask.WithItems) > 0 || ancestorTask.WithParam != ""
			tctx.addOutputsToScope(ctx, resolvedTmpl, varkeys.TasksNodeRef, varkeys.TasksAggregate, ancestor, taskScope, aggregate, true)
		}
		if i := task.Inline; i != nil {
			for _, p := range i.Inputs.Parameters {
				taskScope[varkeys.InputsParameterByName.Concretize(p.Name)] = placeholderGenerator.NextPlaceholder()
			}
		}

		err = addItemsToScope(task.WithItems, task.WithParam, task.WithSequence, taskScope)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "templates.%s.tasks.%s %s", tmpl.Name, task.Name, err.Error())
		}
		err = resolveAllVariables(taskScope, tctx.globalParams, string(taskBytes), workflowTemplateValidation)
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
		_, err = tctx.validateTemplateHolder(ctx, &task, tmplCtx, &task.Arguments, workflowTemplateValidation)
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
	for targetName := range strings.SplitSeq(tmpl.DAG.Target, " ") {
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
func verifyNoCycles(ctx context.Context, tmpl *wfv1.Template, dctx *dagValidationContext) error {
	visited := make(map[string]bool)
	var noCyclesHelper func(taskName string, cycle []string) error
	noCyclesHelper = func(taskName string, cycle []string) error {
		if _, ok := visited[taskName]; ok {
			return nil
		}
		task := dctx.GetTask(ctx, taskName)
		for _, depName := range dctx.GetTaskDependencies(ctx, task.Name) {
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

func sortDAGTasks(ctx context.Context, tmpl *wfv1.Template, tctx *dagValidationContext) error {
	taskMap := make(map[string]*wfv1.DAGTask, len(tmpl.DAG.Tasks))
	sortingGraph := make([]*sorting.TopologicalSortingNode, len(tmpl.DAG.Tasks))
	for index := range tmpl.DAG.Tasks {
		task := tmpl.DAG.Tasks[index]
		taskMap[task.Name] = &task
		dependenciesMap, _ := common.GetTaskDependencies(ctx, &task, tctx)
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
	paramOrArtifactNameRegex = regexp.MustCompile(`^[-a-zA-Z0-9_]+$`)
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
