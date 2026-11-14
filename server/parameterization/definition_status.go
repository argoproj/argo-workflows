package parameterization

import (
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// DefinitionKey never identifies an execution, Secret or provider resource.
// Cluster-scoped keys are used only for native ClusterWorkflowTemplates.
type DefinitionKey struct{ Resource, Name string }

type DefinitionLookup func(DefinitionKey) (*unstructured.Unstructured, error)

func definitionDependencies(o *unstructured.Unstructured) []DefinitionKey {
	var keys []DefinitionKey
	add := func(r, n string) {
		if n != "" {
			keys = append(keys, DefinitionKey{r, n})
		}
	}
	native := func(name string, cluster bool) {
		r := "workflowtemplates"
		if cluster {
			r = "clusterworkflowtemplates"
		}
		add(r, name)
	}
	switch o.GetKind() {
	case "WorkflowForm":
		var x parameterapi.WorkflowFormSpec
		_ = decode(o, &x)
		add("parametertemplates", x.ParameterTemplate)
	case "WorkflowParameterBinding":
		var x Binding
		_ = decode(o, &x)
		add("parametertemplates", x.ParameterTemplate)
		add("workflowforms", x.Form)
		add("workflowactionbindings", x.ActionBinding)
		native(x.WorkflowTemplate, x.ClusterScope)
		for _, input := range x.Inputs {
			add("parametertemplates", input.ParameterTemplate)
			add("workflowforms", input.Form)
		}
	case "ActionDefinition":
		var x ActionDefinition
		_ = decode(o, &x)
		if x.ImplementationRef != nil && staticTemplateName(x.ImplementationRef.Name) {
			native(x.ImplementationRef.Name, x.ImplementationRef.ClusterScope)
		}
	case "WorkflowActionBinding":
		var x ActionBinding
		_ = decode(o, &x)
		native(x.WorkflowTemplate, x.ClusterScope)
		for _, call := range x.Calls {
			add("actiondefinitions", call.ActionDefinition)
			for _, binding := range call.ResultBindings {
				add("actionresulttemplates", binding.ResultTemplate)
			}
		}
	case "WorkflowTemplate", "ClusterWorkflowTemplate":
		var spec wfv1.WorkflowSpec
		if decode(o, &spec) == nil {
			for _, ref := range definitionNativeRefs(&spec) {
				if staticTemplateName(ref.Name) {
					native(ref.Name, ref.ClusterScope)
				}
			}
		}
	}
	return keys
}

// EvaluateDefinition performs static checks over one bounded, namespaced
// reference closure. It never reads values, credentials or external systems.
// Conditions are advisory: preview and submit still validate with caller RBAC.
func EvaluateDefinition(o *unstructured.Unstructured, lookup DefinitionLookup) parameterapi.DefinitionStatus {
	result := parameterapi.DefinitionStatus{ObservedGeneration: o.GetGeneration(), Digest: p.Digest(o.Object["spec"])}
	resolved := map[DefinitionKey]*unstructured.Unstructured{}
	visited := map[DefinitionKey]bool{}
	addError := func(code, path, message string) {
		result.Errors = append(result.Errors, p.FieldError{Code: code, Path: path, Message: message})
	}
	var visit func(*unstructured.Unstructured, int)
	visit = func(current *unstructured.Unstructured, depth int) {
		if depth > 16 || len(visited) > 256 {
			addError("ReferenceLimit", "/spec", "definition reference closure exceeds the supported limit")
			return
		}
		for _, key := range definitionDependencies(current) {
			if visited[key] {
				continue
			}
			if len(visited) >= 256 {
				addError("ReferenceLimit", "/spec", "definition reference closure exceeds the supported limit")
				return
			}
			visited[key] = true
			dependency, err := lookup(key)
			if err != nil || dependency == nil || dependency.GetDeletionTimestamp() != nil {
				addError("ReferenceUnavailable", "/spec", fmt.Sprintf("referenced %s/%s is unavailable", key.Resource, key.Name))
				continue
			}
			resolved[key] = dependency
			result.ResolvedReferences = append(result.ResolvedReferences, parameterapi.ResolvedReference{Kind: dependency.GetKind(), Name: dependency.GetName(), UID: string(dependency.GetUID()), ResourceVersion: dependency.GetResourceVersion(), Digest: p.Digest(dependency.Object["spec"])})
			visit(dependency, depth+1)
		}
	}
	visit(o, 0)
	get := func(resource, name string) *unstructured.Unstructured { return resolved[DefinitionKey{resource, name}] }
	check := func(current *unstructured.Unstructured) {
		if err := checkStaticDefinition(current, get); err != nil {
			addError("InvalidDefinition", "/spec", current.GetKind()+"/"+current.GetName()+": "+err.Error())
		}
	}
	check(o)
	for _, dependency := range resolved {
		check(dependency)
	}
	sort.Slice(result.ResolvedReferences, func(i, j int) bool {
		a, b := result.ResolvedReferences[i], result.ResolvedReferences[j]
		return a.Kind+"/"+a.Name < b.Kind+"/"+b.Name
	})
	sort.Slice(result.Errors, func(i, j int) bool {
		a, b := result.Errors[i], result.Errors[j]
		return a.Code+a.Path+a.Message < b.Code+b.Path+b.Message
	})
	condition := metav1.Condition{Type: "DefinitionValid", ObservedGeneration: o.GetGeneration(), Status: metav1.ConditionTrue, Reason: "StaticChecksPassed", Message: "Supported static schema and reference checks passed; input validation and execution authorization are required separately."}
	if o.GetKind() == "WorkflowForm" {
		var form parameterapi.WorkflowFormSpec
		_ = decode(o, &form)
		if form.ParameterTemplate == "" {
			condition.Status = metav1.ConditionUnknown
			condition.Reason = "BindingContextRequired"
			condition.Message = "This reusable form is validated against a parameter model when referenced by a binding."
		}
	}
	if len(result.Errors) > 0 {
		condition.Status = metav1.ConditionFalse
		condition.Reason = "DefinitionInvalid"
		condition.Message = "Definition or referenced contracts failed static validation; see status.errors."
	}
	result.Conditions = []metav1.Condition{condition}
	return result
}

func checkStaticDefinition(o *unstructured.Unstructured, get func(string, string) *unstructured.Unstructured) error {
	model := func(name string) (*p.Model, error) {
		x := get("parametertemplates", name)
		if x == nil {
			return nil, fmt.Errorf("parameter model is unavailable")
		}
		var m p.Model
		if err := decode(x, &m); err != nil {
			return nil, fmt.Errorf("invalid parameter model structure")
		}
		return &m, nil
	}
	workflow := func(name string, cluster bool) (*wfv1.WorkflowSpec, error) {
		r := "workflowtemplates"
		if cluster {
			r = "clusterworkflowtemplates"
		}
		x := get(r, name)
		if x == nil {
			return nil, fmt.Errorf("native workflow template is unavailable")
		}
		var spec wfv1.WorkflowSpec
		if err := decode(x, &spec); err != nil {
			return nil, fmt.Errorf("invalid native workflow template structure")
		}
		return &spec, nil
	}
	form := func(name string, m p.Model, template string) error {
		x := get("workflowforms", name)
		if x == nil {
			return fmt.Errorf("form is unavailable")
		}
		var f parameterapi.WorkflowFormSpec
		if err := decode(x, &f); err != nil {
			return fmt.Errorf("invalid form structure")
		}
		return validateForm(f, m, template)
	}
	switch o.GetKind() {
	case "WorkflowTemplate", "ClusterWorkflowTemplate":
		var spec wfv1.WorkflowSpec
		if decode(o, &spec) != nil {
			return fmt.Errorf("invalid native template structure")
		}
		if spec.WorkflowTemplateRef != nil {
			return fmt.Errorf("bound native templates cannot redirect their Workflow spec")
		}
		for _, ref := range definitionNativeRefs(&spec) {
			if !staticTemplateName(ref.Name) || !staticTemplateName(ref.Template) {
				return fmt.Errorf("dynamic template references cannot be frozen")
			}
			target, err := workflow(ref.Name, ref.ClusterScope)
			if err != nil {
				return err
			}
			if definitionTemplate(target, ref.Template) == nil {
				return fmt.Errorf("referenced native template is absent")
			}
		}
	case "ParameterTemplate":
		var m p.Model
		if err := decode(o, &m); err != nil {
			return fmt.Errorf("invalid parameter model structure")
		}
		if errors := p.ValidateModelSchema(m); len(errors) > 0 {
			return fmt.Errorf("%s at %s", errors[0].Code, errors[0].Path)
		}
		for _, path := range m.SensitivePaths {
			if formSchemaAt(m.Schema, path) == nil {
				return fmt.Errorf("sensitive path is outside the model")
			}
		}
		for path, schema := range m.SecretSchemas {
			if len(p.ValidateModelSchema(p.Model{Dialect: m.Dialect, Schema: schema})) > 0 {
				return fmt.Errorf("invalid secret value schema at %s", path)
			}
		}
	case "WorkflowForm":
		var f parameterapi.WorkflowFormSpec
		if err := decode(o, &f); err != nil {
			return fmt.Errorf("invalid form structure")
		}
		if f.ParameterTemplate == "" {
			return nil
		}
		m, err := model(f.ParameterTemplate)
		if err != nil {
			return err
		}
		return validateForm(f, *m, f.ParameterTemplate)
	case "ActionResultTemplate":
		var spec parameterapi.ActionResultTemplateSpec
		if err := decode(o, &spec); err != nil {
			return fmt.Errorf("invalid result template structure")
		}
		// Source slots are validated against the actual action by its binding.
		template := &wfv1.Template{}
		for _, name := range resultSourceOutputs(spec.Source) {
			if !staticTemplateName(name) {
				return fmt.Errorf("invalid result output slot")
			}
			template.Outputs.Parameters = append(template.Outputs.Parameters, wfv1.Parameter{Name: name})
		}
		for _, name := range resultSourceInputs(spec.Source) {
			if !staticTemplateName(name) {
				return fmt.Errorf("invalid result input slot")
			}
			template.Inputs.Parameters = append(template.Inputs.Parameters, wfv1.Parameter{Name: name})
		}
		return validateActionResults([]parameterapi.ActionResultDefinition{{Name: "result", ActionResultTemplateSpec: spec}}, template)
	case "ActionDefinition":
		var a ActionDefinition
		if err := decode(o, &a); err != nil {
			return fmt.Errorf("invalid action structure")
		}
		_, err := resolveAction(a, func(ref *wfv1.TemplateRef) (*wfv1.Template, error) {
			spec, err := workflow(ref.Name, ref.ClusterScope)
			if err != nil {
				return nil, err
			}
			return actionTemplate(&wfv1.WorkflowTemplate{Spec: *spec}, ref)
		})
		return err
	case "WorkflowActionBinding":
		var a ActionBinding
		if err := decode(o, &a); err != nil {
			return fmt.Errorf("invalid action binding structure")
		}
		w, err := workflow(a.WorkflowTemplate, a.ClusterScope)
		if err != nil {
			return err
		}
		if err = checkActionCalls(a.Calls); err != nil {
			return err
		}
		for _, call := range a.Calls {
			baseline := definitionTemplate(w, call.Template)
			if len(call.CallPath) > 0 {
				baseline, err = actionCallBaseline(w, call.CallPath, func(ref *wfv1.TemplateRef) (*wfv1.WorkflowSpec, error) { return workflow(ref.Name, ref.ClusterScope) })
				if err != nil {
					return err
				}
			}
			if baseline == nil {
				return fmt.Errorf("action call template is absent")
			}
			x := get("actiondefinitions", call.ActionDefinition)
			if x == nil {
				return fmt.Errorf("action definition is unavailable")
			}
			var def ActionDefinition
			if err = decode(x, &def); err != nil {
				return err
			}
			def, err = resolveAction(def, func(ref *wfv1.TemplateRef) (*wfv1.Template, error) {
				spec, e := workflow(ref.Name, ref.ClusterScope)
				if e != nil {
					return nil, e
				}
				return actionTemplate(&wfv1.WorkflowTemplate{Spec: *spec}, ref)
			})
			if err != nil {
				return err
			}
			if len(call.CallPath) > 0 {
				if _, err = actionAdapter(baseline, call, def.Template, "validate"); err != nil {
					return err
				}
			}
			if _, err = bindActionResults(call, def.Template, func(name string) (*parameterapi.ActionResultTemplateSpec, error) {
				obj := get("actionresulttemplates", name)
				if obj == nil {
					return nil, fmt.Errorf("result template is unavailable")
				}
				var spec parameterapi.ActionResultTemplateSpec
				if resultErr := decode(obj, &spec); resultErr != nil {
					return nil, resultErr
				}
				return &spec, nil
			}); err != nil {
				return err
			}
			if err = checkActionReplacement(baseline, def, call); err != nil {
				return err
			}
		}
	case "WorkflowParameterBinding":
		var b Binding
		if err := decode(o, &b); err != nil {
			return fmt.Errorf("invalid parameter binding structure")
		}
		m, err := model(b.ParameterTemplate)
		if err != nil {
			return err
		}
		w, err := workflow(b.WorkflowTemplate, b.ClusterScope)
		if err != nil {
			return err
		}
		if err = checkStaticMappings(*m, b.Mappings, w.Arguments.Parameters); err != nil {
			return err
		}
		if b.Form != "" {
			if err = form(b.Form, *m, b.ParameterTemplate); err != nil {
				return err
			}
		}
		if b.ActionBinding != "" {
			x := get("workflowactionbindings", b.ActionBinding)
			if x == nil {
				return fmt.Errorf("action binding is unavailable")
			}
			var a ActionBinding
			if err = decode(x, &a); err != nil {
				return fmt.Errorf("invalid action binding structure")
			}
			if a.WorkflowTemplate != b.WorkflowTemplate || a.ClusterScope != b.ClusterScope {
				return fmt.Errorf("action binding targets a different native template")
			}
		}
		seen := map[string]bool{}
		for _, input := range b.Inputs {
			if seen[input.Template] {
				return fmt.Errorf("duplicate input binding")
			}
			seen[input.Template] = true
			t := definitionTemplate(w, input.Template)
			if t == nil || t.Suspend == nil {
				return fmt.Errorf("input form requires a native suspend template")
			}
			im, err := model(input.ParameterTemplate)
			if err != nil {
				return err
			}
			var outputs []wfv1.Parameter
			for _, out := range t.Outputs.Parameters {
				if out.ValueFrom != nil && out.ValueFrom.Supplied != nil {
					outputs = append(outputs, out)
				}
			}
			if len(outputs) != len(input.Mappings) {
				return fmt.Errorf("supplied outputs require complete input mappings")
			}
			if err = checkStaticMappings(*im, input.Mappings, outputs); err != nil {
				return err
			}
			if input.Form != "" {
				if err = form(input.Form, *im, input.ParameterTemplate); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Walk native calls only, never a resource manifest or arbitrary user document.
func definitionNativeRefs(spec *wfv1.WorkflowSpec) []*wfv1.TemplateRef {
	var result []*wfv1.TemplateRef
	add := func(ref *wfv1.TemplateRef) {
		if ref != nil {
			result = append(result, ref)
		}
	}
	hooks := func(hooks wfv1.LifecycleHooks) {
		for _, hook := range hooks {
			add(hook.TemplateRef)
		}
	}
	var visit func(*wfv1.Template, int)
	visit = func(t *wfv1.Template, depth int) {
		if depth > 32 {
			return
		}
		for _, group := range t.Steps {
			for _, step := range group.Steps {
				add(step.TemplateRef)
				hooks(step.Hooks)
				if step.Inline != nil {
					visit(step.Inline, depth+1)
				}
			}
		}
		if t.DAG != nil {
			for _, task := range t.DAG.Tasks {
				add(task.TemplateRef)
				hooks(task.Hooks)
				if task.Inline != nil {
					visit(task.Inline, depth+1)
				}
			}
		}
	}
	for i := range spec.Templates {
		visit(&spec.Templates[i], 0)
	}
	hooks(spec.Hooks)
	return result
}

func checkStaticMappings(model p.Model, mappings []p.Mapping, targets []wfv1.Parameter) error {
	seen := map[string]bool{}
	names := map[string]bool{}
	for _, target := range targets {
		names[target.Name] = true
	}
	for _, mapping := range mappings {
		if seen[mapping.Parameter] || !names[mapping.Parameter] {
			return fmt.Errorf("mapping target is absent or duplicated")
		}
		seen[mapping.Parameter] = true
		schema := formSchemaAt(model.Schema, mapping.Path)
		if schema == nil {
			return fmt.Errorf("mapping path is outside the model")
		}
		for _, secret := range model.SensitivePaths {
			if p.PathsOverlap(mapping.Path, secret) {
				if mapping.Encoding != "secret-reference-json" || len(strings.Split(mapping.Path, "/")) > len(strings.Split(secret, "/")) {
					return fmt.Errorf("sensitive input requires complete reference-only JSON")
				}
			}
		}
		typ, _ := schema["type"].(string)
		compatible := mapping.Encoding == "json" || mapping.Encoding == "secret-reference-json" ||
			(mapping.Encoding == "decimal" && (typ == "number" || typ == "integer")) ||
			(mapping.Encoding == "string" && typ == "string") || (mapping.Encoding == "boolean" && typ == "boolean")
		if !compatible {
			return fmt.Errorf("mapping encoding is incompatible with its schema")
		}
	}
	return nil
}

func definitionTemplate(spec *wfv1.WorkflowSpec, name string) *wfv1.Template {
	for i := range spec.Templates {
		if spec.Templates[i].Name == name {
			return &spec.Templates[i]
		}
	}
	return nil
}
