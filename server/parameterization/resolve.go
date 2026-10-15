package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

type Binding = parameterapi.WorkflowParameterBindingSpec
type Preview struct {
	Definitions        *parameterapi.ExecutionDefinitions `json:"definitions,omitempty"`
	Revision           string                             `json:"revision"`
	Binding            string                             `json:"binding"`
	ExecutionDigest    string                             `json:"executionDigest"`
	Workflow           *wfv1.Workflow                     `json:"workflow"`
	Errors             []p.FieldError                     `json:"errors"`
	DefinitionVersions map[string]string                  `json:"definitionVersions"`
	InputForms         map[string]FrozenInputForm         `json:"inputForms,omitempty"`
}

func (s Store) Preview(ctx context.Context, bindingName, revisionName string) (*Preview, error) {
	ro, err := s.get(ctx, "parameterrevisions", revisionName)
	if err != nil {
		return nil, err
	}
	var rev Revision
	if err = decode(ro, &rev); err != nil {
		return nil, err
	}
	bo, err := s.get(ctx, "workflowparameterbindings", bindingName)
	if err != nil {
		return nil, err
	}
	var binding Binding
	if err = decode(bo, &binding); err != nil {
		return nil, err
	}
	if binding.ParameterTemplate != rev.Template {
		return nil, fmt.Errorf("revision parameter model does not match binding")
	}
	values, _, errors := p.Resolve(rev.Model, rev.Values)
	result := &Preview{Revision: revisionName, Binding: bindingName, Errors: errors, Definitions: &parameterapi.ExecutionDefinitions{ParameterBinding: binding, Actions: map[string]parameterapi.ActionDefinitionSpec{}}}
	if len(errors) > 0 {
		return result, nil
	}
	versions := map[string]string{}
	if err = s.resolveLatestSecrets(ctx, rev.Model, values, versions); err != nil {
		return nil, err
	}
	args, err := p.Arguments(rev.Model, values, binding.Mappings)
	if err != nil {
		return nil, err
	}
	var sourceSpec *wfv1.WorkflowSpec
	var sourceVersion string
	if binding.ClusterScope {
		source, sourceErr := auth.GetWfClient(ctx).ArgoprojV1alpha1().ClusterWorkflowTemplates().Get(ctx, binding.WorkflowTemplate, metav1.GetOptions{})
		if sourceErr != nil {
			return nil, sourceErr
		}
		sourceSpec = source.Spec.DeepCopy()
		sourceVersion = "ClusterWorkflowTemplate/" + string(source.UID) + ":" + source.ResourceVersion
	} else {
		source, sourceErr := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowTemplates(s.Namespace).Get(ctx, binding.WorkflowTemplate, metav1.GetOptions{})
		if sourceErr != nil {
			return nil, sourceErr
		}
		sourceSpec = source.Spec.DeepCopy()
		sourceVersion = string(source.UID) + ":" + source.ResourceVersion
	}
	wf := &wfv1.Workflow{TypeMeta: metav1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"}, ObjectMeta: metav1.ObjectMeta{Namespace: s.Namespace}, Spec: *sourceSpec}
	// Keep native template execution history discoverable after freezing the
	// source into this Workflow. Labels identify the source, not a live binding.
	sourceLabel := common.LabelKeyWorkflowTemplate
	if binding.ClusterScope {
		sourceLabel = common.LabelKeyClusterWorkflowTemplate
	}
	wf.Labels = map[string]string{sourceLabel: binding.WorkflowTemplate}
	var preciseCalls []ActionCall
	resolvedActions := map[string]*wfv1.Template{}
	resolvedDefinitions := map[string]ActionDefinition{}
	if binding.ActionBinding != "" {
		ao, bindingErr := s.get(ctx, "workflowactionbindings", binding.ActionBinding)
		if bindingErr != nil {
			return nil, bindingErr
		}
		var actionBinding ActionBinding
		if bindingErr = decode(ao, &actionBinding); bindingErr != nil {
			return nil, bindingErr
		}
		if actionBinding.WorkflowTemplate != binding.WorkflowTemplate || actionBinding.ClusterScope != binding.ClusterScope {
			return nil, fmt.Errorf("action binding targets a different WorkflowTemplate")
		}
		result.Definitions.ActionBinding = &actionBinding
		versions["actionBinding"] = definitionContentVersion(ao)
		if err = checkActionCalls(actionBinding.Calls); err != nil {
			return nil, err
		}
		implementations := &templateClosure{namespace: s.Namespace, versions: versions, sources: map[string]wfv1.TemplateHolder{}}
		for _, call := range actionBinding.Calls {
			action, actionErr := s.get(ctx, "actiondefinitions", call.ActionDefinition)
			if actionErr != nil {
				return nil, actionErr
			}
			var def ActionDefinition
			if actionErr = decode(action, &def); actionErr != nil {
				return nil, actionErr
			}
			var frozen parameterapi.ActionDefinitionSpec
			if actionErr = decode(action, &frozen); actionErr != nil {
				return nil, actionErr
			}
			result.Definitions.Actions[call.ActionDefinition] = frozen
			def, actionErr = resolveAction(def, func(ref *wfv1.TemplateRef) (*wfv1.Template, error) {
				key, sourceErr := implementations.source(ctx, ref)
				if sourceErr != nil {
					return nil, sourceErr
				}
				return actionTemplate(implementations.sources[key], ref)
			})
			if actionErr != nil {
				return nil, actionErr
			}
			resolvedDefinitions[call.ActionDefinition] = def
			if len(call.ResultBindings) > 0 {
				if result.Definitions.CallResults == nil {
					result.Definitions.CallResults = map[string][]parameterapi.ActionResultDefinition{}
					result.Definitions.ResultTemplates = map[string]parameterapi.ActionResultTemplateSpec{}
				}
				results, bindErr := bindActionResults(call, def.Template, func(name string) (*parameterapi.ActionResultTemplateSpec, error) {
					if cached, ok := result.Definitions.ResultTemplates[name]; ok {
						return cached.DeepCopy(), nil
					}
					obj, resultErr := s.get(ctx, "actionresulttemplates", name)
					if resultErr != nil {
						return nil, resultErr
					}
					var spec parameterapi.ActionResultTemplateSpec
					if resultErr = decode(obj, &spec); resultErr != nil {
						return nil, resultErr
					}
					result.Definitions.ResultTemplates[name] = spec
					versions["result/"+name] = definitionContentVersion(obj)
					return &spec, nil
				})
				if bindErr != nil {
					return nil, bindErr
				}
				result.Definitions.CallResults[resultCallKey(call)] = results
			}
			implementation := def.Template.DeepCopy()
			implementation.Name = call.Template
			if implementation.Metadata.Annotations == nil {
				implementation.Metadata.Annotations = map[string]string{}
			}
			implementation.Metadata.Annotations[p.InputContractAnnotation], actionErr = p.FreezeActionContract(def.InputSchema)
			if actionErr != nil {
				return nil, actionErr
			}
			implementation.Metadata.Annotations[p.OutputContractAnnotation], actionErr = p.FreezeActionContract(def.OutputSchema)
			if actionErr != nil {
				return nil, actionErr
			}
			if len(call.CallPath) > 0 {
				preciseCalls = append(preciseCalls, call)
				resolvedActions[call.ActionDefinition] = implementation
			} else {
				index := -1
				for i, t := range wf.Spec.Templates {
					if t.Name == call.Template {
						index = i
					}
				}
				if index < 0 {
					return nil, fmt.Errorf("action call template %s is absent", call.Template)
				}
				if actionErr = checkActionReplacement(&wf.Spec.Templates[index], def, call); actionErr != nil {
					return nil, actionErr
				}
				wf.Spec.Templates[index] = *implementation
			}
			versions["action/"+call.ActionDefinition] = definitionContentVersion(action)
		}
	}
	if binding.Form != "" {
		form, formErr := s.get(ctx, "workflowforms", binding.Form)
		if formErr != nil {
			return nil, formErr
		}
		versions["form"] = definitionContentVersion(form)
		var declared parameterapi.WorkflowFormSpec
		if formErr = decode(form, &declared); formErr != nil {
			return nil, formErr
		}
		if formErr = validateForm(declared, rev.Model, rev.Template); formErr != nil {
			return nil, formErr
		}
		result.Definitions.Form = &declared
		sources, formErr := formOptions(declared)
		if formErr != nil {
			return nil, formErr
		}
		for path, source := range expandFormOptions(sources, values) {
			value, exists := p.Pointer(values, path)
			if !exists {
				continue // Required values and defaults have already been resolved by the model.
			}
			options, queryErr := s.Options(ctx, OptionsRequest{Binding: bindingName, Path: path, Values: values}, s.OptionProviders)
			if queryErr != nil {
				return nil, queryErr
			}
			if !source.AllowManual {
				found := false
				for _, option := range options.Items {
					if !option.Disabled && p.Digest(option.Value) == p.Digest(value) {
						found = true
						break
					}
				}
				if !found {
					result.Errors = append(result.Errors, p.FieldError{Code: "OptionUnavailable", Path: path, Message: "selected value is no longer available"})
					return result, nil
				}
			}
			versions["options/"+path] = options.Version + ":" + options.ContextDigest
		}
	}
	if secretErr := s.authorizeMappedSecrets(ctx, rev.Model, values, binding.Mappings, versions); secretErr != nil {
		return nil, secretErr
	}
	for _, secret := range binding.Secrets {
		declared := false
		for _, path := range rev.Model.SensitivePaths {
			if p.MatchPath(path, secret.Path) {
				declared = true
			}
		}
		if !declared {
			return nil, fmt.Errorf("secret binding path must be declared sensitive")
		}
		raw, exists := p.Pointer(values, secret.Path)
		if !exists || raw == nil {
			return nil, fmt.Errorf("secret reference is absent")
		}
		obj, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid secret reference")
		}
		ref, ok := obj["secretRef"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid secret reference")
		}
		name, key := ref["name"].(string), ref["key"].(string)
		credential, secretErr := auth.GetKubeClient(ctx).CoreV1().Secrets(s.Namespace).Get(ctx, name, metav1.GetOptions{})
		if secretErr != nil {
			return nil, fmt.Errorf("secret reference is unavailable or forbidden")
		}
		if leaseErr := p.ValidateExternalSecretLease(credential, time.Now()); leaseErr != nil {
			return nil, leaseErr
		}
		if credential.Immutable == nil || !*credential.Immutable {
			return nil, fmt.Errorf("reproducible submission requires an immutable versioned Secret")
		}
		if _, ok := credential.Data[key]; !ok {
			return nil, fmt.Errorf("secret key is absent")
		}
		if secretLogicalSchema(rev.Model, secret.Path) != nil {
			if valueErr := validateSecretValue(rev.Model, secret.Path, string(credential.Data[key])); valueErr != nil {
				return nil, valueErr
			}
		}
		matched := false
		for i := range wf.Spec.Templates {
			t := &wf.Spec.Templates[i]
			if t.Name != secret.Template {
				continue
			}
			if t.Container == nil {
				return nil, fmt.Errorf("this Secret binding requires a native container template")
			}
			if secret.Environment == "" {
				return nil, fmt.Errorf("secret environment name is required")
			}
			for _, e := range t.Container.Env {
				if e.Name == secret.Environment {
					return nil, fmt.Errorf("secret binding collides with an existing environment variable")
				}
			}
			t.Container.Env = append(t.Container.Env, corev1.EnvVar{Name: secret.Environment, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: key}}})
			matched = true
		}
		if !matched {
			return nil, fmt.Errorf("secret target template is absent")
		}
		versions["secret/"+name] = string(credential.UID) + ":" + credential.ResourceVersion
	}
	inputForms, err := s.freezeInputForms(ctx, binding.Inputs, wf, versions)
	if err != nil {
		return nil, err
	}
	result.InputForms = inputForms
	if err := freezeTemplateClosure(ctx, s.Namespace, wf, versions); err != nil {
		return nil, err
	}
	for _, call := range preciseCalls {
		baseline, err := actionCallBaseline(&wf.Spec, call.CallPath, nil)
		if err != nil {
			return nil, err
		}
		if err = checkActionReplacement(baseline, resolvedDefinitions[call.ActionDefinition], call); err != nil {
			return nil, err
		}
		for _, secret := range binding.Secrets {
			if secret.Template == baseline.Name {
				return nil, fmt.Errorf("callPath action conflicts with a template-level Secret binding; use explicit action inputs")
			}
		}
		for _, input := range binding.Inputs {
			if input.Template == baseline.Name {
				return nil, fmt.Errorf("callPath action conflicts with a runtime input form")
			}
		}
	}
	if err := compileActionCalls(wf, preciseCalls, resolvedActions); err != nil {
		return nil, err
	}
	declared := map[string]bool{}
	for _, v := range wf.Spec.Arguments.Parameters {
		declared[v.Name] = true
	}
	for name := range args {
		if !declared[name] {
			return nil, fmt.Errorf("unknown workflow parameter %s", name)
		}
	}
	for i, v := range wf.Spec.Arguments.Parameters {
		if value, ok := args[v.Name]; ok {
			wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(value)
			wf.Spec.Arguments.Parameters[i].ValueFrom = nil
		}
	}
	result.Workflow = wf
	result.DefinitionVersions = map[string]string{"binding": definitionContentVersion(bo), "workflowTemplate": sourceVersion, "model": p.Digest(rev.Model)}
	maps.Copy(result.DefinitionVersions, versions)
	result.Definitions.Versions = result.DefinitionVersions
	result.ExecutionDigest = p.Digest(map[string]any{"spec": wf.Spec, "revision": revisionName, "definitions": result.DefinitionVersions, "inputForms": result.InputForms})
	return result, nil
}

// ChangedArguments is used by retry impact analysis; absent values are distinct from empty strings.
func ChangedArguments(a, b []wfv1.Parameter) []string {
	old := map[string]string{}
	next := map[string]string{}
	for _, x := range a {
		raw, _ := json.Marshal(x)
		old[x.Name] = string(raw)
	}
	for _, x := range b {
		raw, _ := json.Marshal(x)
		next[x.Name] = string(raw)
	}
	keys := map[string]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range next {
		keys[k] = true
	}
	var result []string
	for k := range keys {
		if old[k] != next[k] {
			result = append(result, k)
		}
	}
	sort.Strings(result)
	return result
}

// ActionDefinition describes a typed native template implementation, not a second execution engine.
type ActionDefinition struct {
	Results           []parameterapi.ActionResultDefinition `json:"results,omitempty"`
	ExecutionContract *parameterapi.ActionExecutionContract `json:"executionContract,omitempty"`
	InputSchema       map[string]any                        `json:"inputSchema"`
	OutputSchema      map[string]any                        `json:"outputSchema"`
	Template          *wfv1.Template                        `json:"template,omitempty"`
	ImplementationRef *wfv1.TemplateRef                     `json:"implementationRef,omitempty"`
}
type ActionCall = parameterapi.ActionCall
type ActionBinding = parameterapi.WorkflowActionBindingSpec

func validateAction(def ActionDefinition) error {
	if def.Template == nil {
		return fmt.Errorf("action implementation must be resolved before validation")
	}
	for _, s := range []map[string]any{def.InputSchema, def.OutputSchema} {
		if s["type"] != "object" || len(p.CheckSchema(s, "")) > 0 {
			return fmt.Errorf("action requires supported object input/output schemas")
		}
	}
	for _, contract := range []struct {
		schema     map[string]any
		parameters []wfv1.Parameter
		output     bool
	}{{def.InputSchema, def.Template.Inputs.Parameters, false}, {def.OutputSchema, def.Template.Outputs.Parameters, true}} {
		props, _ := contract.schema["properties"].(map[string]any)
		names := map[string]bool{}
		for _, param := range contract.parameters {
			names[param.Name] = true
			if _, ok := props[param.Name]; !ok {
				return fmt.Errorf("native action parameter is outside the declared contract")
			}
		}
		for name := range props {
			if name == "result" && (def.Template.HTTP != nil || def.Template.Script != nil) && !names[name] && contract.output {
				continue
			}
			if !names[name] {
				return fmt.Errorf("action contract parameter %s is absent from native implementation", name)
			}
		}
	}
	if def.Template.DAG != nil || len(def.Template.Steps) > 0 || def.Template.Memoize != nil || def.Template.ContainerSet != nil || def.Template.Plugin != nil || def.Template.IsDaemon() {
		return fmt.Errorf("action contract requires a supported non-daemon leaf implementation without memoization")
	}
	for _, contract := range []map[string]any{def.InputSchema, def.OutputSchema} {
		if _, err := p.FreezeActionContract(contract); err != nil {
			return err
		}
	}
	switch def.Template.GetType() {
	case wfv1.TemplateTypeContainer, wfv1.TemplateTypeScript, wfv1.TemplateTypeResource, wfv1.TemplateTypeHTTP:
	default:
		return fmt.Errorf("action requires a supported container, script, resource or HTTP implementation")
	}
	// Raw templates are subject to normal Argo validation and admission on submission.
	if strings.TrimSpace(def.Template.Name) == "" {
		return fmt.Errorf("action implementation name is required")
	}
	if err := validateActionResults(def.Results, def.Template); err != nil {
		return err
	}
	return validateActionExecutionContract(def.ExecutionContract, def.Template)
}

type SecretBinding = parameterapi.SecretBinding
