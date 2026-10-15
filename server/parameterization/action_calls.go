package parameterization

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

const bindableCallsAnnotation = p.Group + "/bindable-calls"

// A binding never owns scheduling. This compiler only replaces an opted-in call
// target in a private native snapshot, cloning shared ancestors by named path.
type nativeActionSite struct {
	template *string
	ref      **wfv1.TemplateRef
	inline   *wfv1.Template
	loop     bool
}

func actionSite(parent *wfv1.Template, name string) (*nativeActionSite, error) {
	var site *nativeActionSite
	count := 0
	if parent.DAG != nil {
		for i := range parent.DAG.Tasks {
			t := &parent.DAG.Tasks[i]
			if t.Name == name {
				count++
				site = &nativeActionSite{&t.Template, &t.TemplateRef, t.Inline, len(t.WithItems) > 0 || t.WithParam != "" || t.WithSequence != nil}
			}
		}
	}
	for i := range parent.Steps {
		for j := range parent.Steps[i].Steps {
			t := &parent.Steps[i].Steps[j]
			if t.Name == name {
				count++
				site = &nativeActionSite{&t.Template, &t.TemplateRef, t.Inline, len(t.WithItems) > 0 || t.WithParam != "" || t.WithSequence != nil}
			}
		}
	}
	if count != 1 || site.inline != nil || (*site.template == "") == (*site.ref == nil) {
		return nil, fmt.Errorf("action call %s must identify one named native template call", name)
	}
	return site, nil
}

func checkActionCalls(calls []ActionCall) error {
	seen := map[string]bool{}
	legacy, precise := false, false
	for _, call := range calls {
		if (call.Template == "") == (len(call.CallPath) == 0) {
			return fmt.Errorf("exactly one of template or callPath is required")
		}
		if call.Template != "" {
			legacy = true
			if len(call.InputMappings) > 0 || len(call.OutputMappings) > 0 || len(call.InputArtifactMappings) > 0 || len(call.OutputArtifactMappings) > 0 {
				return fmt.Errorf("interface mappings require a callPath")
			}
			if seen[call.Template] {
				return fmt.Errorf("duplicate action binding for template %s", call.Template)
			}
			seen[call.Template] = true
			continue
		}
		precise = true
		if len(call.CallPath) < 2 || len(call.CallPath) > 32 {
			return fmt.Errorf("action callPath requires 2 to 32 named segments")
		}
		for _, name := range call.CallPath {
			if !staticTemplateName(name) || strings.ContainsAny(name, "/.[ ]") {
				return fmt.Errorf("action callPath requires static native names")
			}
		}
		key := strings.Join(call.CallPath, "/")
		for old := range seen {
			if key == old || strings.HasPrefix(key, old+"/") || strings.HasPrefix(old, key+"/") {
				return fmt.Errorf("duplicate or overlapping action callPath")
			}
		}
		seen[key] = true
	}
	if legacy && precise {
		return fmt.Errorf("legacy template bindings and callPath bindings cannot be mixed")
	}
	return nil
}

func authorizeActionSite(parent *wfv1.Template, name string) error {
	var allowed []string
	if json.Unmarshal([]byte(parent.Metadata.Annotations[bindableCallsAnnotation]), &allowed) != nil {
		return fmt.Errorf("action call %s is not author-approved for binding", name)
	}
	if slices.Contains(allowed, name) {
		return nil
	}
	return fmt.Errorf("action call %s is not author-approved for binding", name)
}

func actionCallBaseline(spec *wfv1.WorkflowSpec, path []string, lookup func(*wfv1.TemplateRef) (*wfv1.WorkflowSpec, error)) (*wfv1.Template, error) {
	if path[0] != spec.Entrypoint {
		return nil, fmt.Errorf("action callPath must start at the native entrypoint")
	}
	parent := definitionTemplate(spec, path[0])
	visited := map[*wfv1.Template]bool{}
	for i, name := range path[1:] {
		if parent == nil || visited[parent] {
			return nil, fmt.Errorf("recursive or missing action call ancestor")
		}
		visited[parent] = true
		site, err := actionSite(parent, name)
		if err != nil {
			return nil, err
		}
		last := i == len(path)-2
		if last {
			if err = authorizeActionSite(parent, name); err != nil {
				return nil, err
			}
		} else if site.loop {
			return nil, fmt.Errorf("action callPath cannot descend through a loop; bind its named leaf site instead")
		}
		if *site.ref != nil {
			if lookup == nil {
				return nil, fmt.Errorf("action call reference closure is not frozen")
			}
			spec, err = lookup(*site.ref)
			if err != nil {
				return nil, err
			}
			parent = definitionTemplate(spec, (*site.ref).Template)
		} else {
			parent = definitionTemplate(spec, *site.template)
		}
	}
	if parent == nil {
		return nil, fmt.Errorf("action baseline template is absent")
	}
	return parent, nil
}

func actionParameterNames(params []wfv1.Parameter) []string {
	names := make([]string, 0, len(params))
	for _, param := range params {
		names = append(names, param.Name)
	}
	return names
}

func completeActionMappings(mappings []parameterapi.ActionParameterMapping, from, to []string) ([]parameterapi.ActionParameterMapping, error) {
	if len(mappings) == 0 {
		for _, name := range from {
			mappings = append(mappings, parameterapi.ActionParameterMapping{From: name, To: name})
		}
	}
	src, dst := map[string]bool{}, map[string]bool{}
	for _, name := range from {
		if src[name] {
			return nil, fmt.Errorf("duplicate baseline interface parameter")
		}
		src[name] = true
	}
	for _, name := range to {
		if dst[name] {
			return nil, fmt.Errorf("duplicate action interface parameter")
		}
		dst[name] = true
	}
	for _, m := range mappings {
		if !src[m.From] || !dst[m.To] {
			return nil, fmt.Errorf("action mapping is unknown, duplicated or incomplete")
		}
		delete(src, m.From)
		delete(dst, m.To)
	}
	if len(src) > 0 || len(dst) > 0 {
		return nil, fmt.Errorf("action mapping is incomplete")
	}
	return mappings, nil
}

func actionAdapter(baseline *wfv1.Template, call ActionCall, implementation *wfv1.Template, name string) (*wfv1.Template, error) {
	// Implicit result/exitCode and daemon/composite semantics cannot be
	// replaced by a native DAG adapter without changing their public meaning.
	if baseline.DAG != nil || len(baseline.Steps) > 0 || baseline.HTTP != nil || baseline.Script != nil || baseline.IsDaemon() || baseline.Memoize != nil || baseline.RetryStrategy != nil || baseline.Outputs.Result != nil || baseline.Outputs.ExitCode != nil {
		return nil, fmt.Errorf("action call baseline requires a native leaf without implicit result, retry or memoization")
	}
	inputs, err := completeActionMappings(call.InputMappings, actionParameterNames(baseline.Inputs.Parameters), actionParameterNames(implementation.Inputs.Parameters))
	if err != nil {
		return nil, err
	}
	outputs := actionParameterNames(implementation.Outputs.Parameters)
	implicitResult := implementation.HTTP != nil || implementation.Script != nil
	if implicitResult {
		outputs = append(outputs, "result")
	}
	out, err := completeActionMappings(call.OutputMappings, outputs, actionParameterNames(baseline.Outputs.Parameters))
	if err != nil {
		return nil, err
	}
	adapter := &wfv1.Template{Name: name, Inputs: *baseline.Inputs.DeepCopy(), DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "invoke", Template: implementation.Name}}}}
	for _, m := range inputs {
		adapter.DAG.Tasks[0].Arguments.Parameters = append(adapter.DAG.Tasks[0].Arguments.Parameters, wfv1.Parameter{Name: m.To, Value: wfv1.AnyStringPtr("{{inputs.parameters." + m.From + "}}")})
	}
	for _, m := range out {
		expr := "{{tasks.invoke.outputs.parameters." + m.From + "}}"
		if implicitResult && m.From == "result" {
			expr = "{{tasks.invoke.outputs.result}}"
		}
		for _, original := range baseline.Outputs.Parameters {
			if original.Name == m.To {
				mapped := *original.DeepCopy()
				mapped.Value = nil
				mapped.ValueFrom = &wfv1.ValueFrom{Parameter: expr}
				adapter.Outputs.Parameters = append(adapter.Outputs.Parameters, mapped)
			}
		}
	}
	if err := mapActionArtifacts(baseline, implementation, call, adapter); err != nil {
		return nil, err
	}
	return adapter, nil
}

func compileActionCalls(wf *wfv1.Workflow, calls []ActionCall, implementations map[string]*wfv1.Template) error {
	if err := checkActionCalls(calls); err != nil {
		return err
	}
	original := wf.Spec.DeepCopy()
	clones := map[string]*wfv1.Template{}
	used := map[string]bool{}
	for _, t := range wf.Spec.Templates {
		used[t.Name] = true
	}
	additions := []*wfv1.Template{}
	add := func(t *wfv1.Template) error {
		if used[t.Name] {
			return fmt.Errorf("action snapshot template name collision")
		}
		used[t.Name] = true
		additions = append(additions, t)
		return nil
	}
	for _, call := range calls {
		if len(call.CallPath) == 0 {
			continue
		}
		baseline, err := actionCallBaseline(original, call.CallPath, nil)
		if err != nil {
			return err
		}
		impl := implementations[call.ActionDefinition].DeepCopy()
		key := strings.Join(call.CallPath, "/")
		impl.Name = "action-" + p.Digest(key)[:24]
		wrapperName := "call-" + p.Digest(key)[:24]
		adapter, err := actionAdapter(baseline, call, impl, wrapperName)
		if err != nil {
			return err
		}
		if err = add(impl); err != nil {
			return err
		}
		if err = add(adapter); err != nil {
			return err
		}
		source := definitionTemplate(original, original.Entrypoint)
		var parent *wfv1.Template
		for i := 0; i < len(call.CallPath)-1; i++ {
			prefix := strings.Join(call.CallPath[:i+1], "/")
			clone := clones[prefix]
			if clone == nil {
				clone = source.DeepCopy()
				clone.Name = "site-" + p.Digest(prefix)[:24]
				clones[prefix] = clone
				if err = add(clone); err != nil {
					return err
				}
			}
			if parent == nil {
				wf.Spec.Entrypoint = clone.Name
			} else {
				site, _ := actionSite(parent, call.CallPath[i])
				*site.template = clone.Name
			}
			parent = clone
			site, siteErr := actionSite(source, call.CallPath[i+1])
			if siteErr != nil {
				return siteErr
			}
			source = definitionTemplate(original, *site.template)
		}
		site, err := actionSite(parent, call.CallPath[len(call.CallPath)-1])
		if err != nil {
			return err
		}
		*site.template = wrapperName
	}
	for _, t := range additions {
		wf.Spec.Templates = append(wf.Spec.Templates, *t)
	}
	return nil
}
