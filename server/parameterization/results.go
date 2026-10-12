package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	api "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

const maxResultBytes = 256 * 1024

func resultCallKey(call api.ActionCall) string {
	if len(call.CallPath) > 0 {
		return "call/" + strings.Join(call.CallPath, "/")
	}
	return "template/" + call.Template
}

func resultSourceOutputs(source api.ActionResultSource) []string {
	if source.Resource != nil {
		if source.Resource.NameOutput == "" && source.Resource.UIDOutput == "" {
			return nil
		}
		return []string{source.Resource.NameOutput, source.Resource.UIDOutput}
	}
	return []string{source.Parameter}
}

func resultSourceInputs(source api.ActionResultSource) []string {
	if source.Resource == nil || source.Resource.NameInput == "" && source.Resource.UIDInput == "" {
		return nil
	}
	return []string{source.Resource.NameInput, source.Resource.UIDInput}
}
func resultSlotMapping(slots []string, mappings []api.ActionParameterMapping) (map[string]string, error) {
	mapped := map[string]string{}
	for _, m := range mappings {
		if !slices.Contains(slots, m.To) || m.From == "" || mapped[m.To] != "" {
			return nil, fmt.Errorf("result mapping slot is absent or duplicated")
		}
		mapped[m.To] = m.From
	}
	return mapped, nil
}

// bindActionResults resolves reusable result contracts independently per native
// call. It performs no external query and cannot redirect the action itself.
func bindActionResults(call api.ActionCall, template *wfv1.Template, lookup func(string) (*api.ActionResultTemplateSpec, error)) ([]api.ActionResultDefinition, error) {
	if len(call.ResultBindings) > 16 {
		return nil, fmt.Errorf("too many result bindings")
	}
	var results []api.ActionResultDefinition
	for _, binding := range call.ResultBindings {
		if len(validation.IsDNS1123Subdomain(binding.ResultTemplate)) != 0 {
			return nil, fmt.Errorf("invalid result template reference")
		}
		spec, err := lookup(binding.ResultTemplate)
		if err != nil {
			return nil, err
		}
		if spec == nil {
			return nil, fmt.Errorf("result template is unavailable")
		}
		r := api.ActionResultDefinition{Name: binding.Name, ActionResultTemplateSpec: *spec.DeepCopy()}
		slots := resultSourceOutputs(r.Source)
		mapped := map[string]string{}
		if len(binding.OutputMappings) > len(slots) {
			return nil, fmt.Errorf("too many result output mappings")
		}
		for _, mapping := range binding.OutputMappings {
			if !slices.Contains(slots, mapping.To) || mapping.From == "" || mapped[mapping.To] != "" {
				return nil, fmt.Errorf("result mapping slot is absent or duplicated")
			}
			mapped[mapping.To] = mapping.From
		}
		name := func(slot string) string {
			if mapped[slot] != "" {
				return mapped[slot]
			}
			return slot
		}
		r.Source.Parameter = name(r.Source.Parameter)
		if r.Source.Resource != nil {
			r.Source.Resource.NameOutput = name(r.Source.Resource.NameOutput)
			r.Source.Resource.UIDOutput = name(r.Source.Resource.UIDOutput)
		}
		inputs, err := resultSlotMapping(resultSourceInputs(r.Source), binding.InputMappings)
		if err != nil {
			return nil, err
		}
		if r.Source.Resource != nil {
			if v := inputs[r.Source.Resource.NameInput]; v != "" {
				r.Source.Resource.NameInput = v
			}
			if v := inputs[r.Source.Resource.UIDInput]; v != "" {
				r.Source.Resource.UIDInput = v
			}
		}
		results = append(results, r)
	}
	return results, validateActionResults(results, template)
}

type NodeResult struct {
	Name       string           `json:"name"`
	Title      string           `json:"title"`
	Scope      string           `json:"scope"`
	Views      []api.ResultView `json:"views"`
	Value      any              `json:"value,omitempty"`
	Available  bool             `json:"available"`
	Error      string           `json:"error,omitempty"`
	ObservedAt string           `json:"observedAt,omitempty"`
	Stale      bool             `json:"stale,omitempty"`
}

type NodeResults struct {
	Workflow       string       `json:"workflow"`
	UID            string       `json:"uid"`
	NodeID         string       `json:"nodeId"`
	Invocation     string       `json:"invocation"`
	ContractDigest string       `json:"contractDigest"`
	History        string       `json:"history,omitempty"`
	Items          []NodeResult `json:"items"`
}

func validResultPointer(s string) bool {
	if len(s) > 1024 || strings.Contains(s, "{{") || (s != "" && !strings.HasPrefix(s, "/")) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '~' {
			i++
			if i >= len(s) || s[i] != '0' && s[i] != '1' {
				return false
			}
		}
	}
	return true
}

func resultSchema(r api.ActionResultDefinition) (*gojsonschema.Schema, error) {
	var value map[string]any
	if len(r.Schema.Raw) > 32768 || json.Unmarshal(r.Schema.Raw, &value) != nil || len(p.CheckSchema(value, "")) != 0 {
		return nil, fmt.Errorf("result requires a supported schema")
	}
	loader := gojsonschema.NewSchemaLoader()
	loader.AutoDetect = false
	loader.Draft = gojsonschema.Draft7
	compiled, err := loader.Compile(gojsonschema.NewGoLoader(value))
	if err != nil {
		return nil, fmt.Errorf("invalid result schema")
	}
	return compiled, nil
}

func validateActionResults(results []api.ActionResultDefinition, t *wfv1.Template) error {
	if len(results) > 16 {
		return fmt.Errorf("too many result definitions")
	}
	names := map[string]bool{}
	inputs := map[string]bool{}
	for _, i := range t.Inputs.Parameters {
		inputs[i.Name] = true
	}
	outputs := map[string]bool{}
	for _, o := range t.Outputs.Parameters {
		outputs[o.Name] = true
	}
	if t.HTTP != nil || t.Script != nil {
		outputs["result"] = true
	}
	for _, r := range results {
		if len(validation.IsDNS1123Label(r.Name)) != 0 || names[r.Name] || len(r.Title) > 256 {
			return fmt.Errorf("invalid or duplicate result name/title")
		}
		names[r.Name] = true
		if (r.Source.Parameter == "") == (r.Source.Resource == nil) || !slices.Contains([]string{"json", "text"}, r.Source.Format) {
			return fmt.Errorf("result requires exactly one supported source and format")
		}
		if r.Source.Parameter != "" && !outputs[r.Source.Parameter] {
			return fmt.Errorf("result source is not a declared native output")
		}
		if s := r.Source.Resource; s != nil {
			outputPair := s.NameOutput != "" && s.UIDOutput != "" && s.NameOutput != s.UIDOutput && outputs[s.NameOutput] && outputs[s.UIDOutput] && s.NameInput == "" && s.UIDInput == ""
			inputPair := s.NameInput != "" && s.UIDInput != "" && s.NameInput != s.UIDInput && inputs[s.NameInput] && inputs[s.UIDInput] && s.NameOutput == "" && s.UIDOutput == ""
			if (!outputPair && !inputPair) || len(validation.IsDNS1123Label(s.Resource)) != 0 || len(validation.IsDNS1123Label(s.Version)) != 0 {
				return fmt.Errorf("resource result requires static type and exactly one native name/UID input or output pair")
			}
			// Only custom resource status and produced ConfigMap data are public
			// sources. No Secret, metadata, arbitrary namespace or subresource proxy.
			prefix := "/status/"
			if s.Group == "" {
				if s.Resource != "configmaps" {
					return fmt.Errorf("core result sources only support ConfigMaps")
				}
				prefix = "/data/"
			} else if len(validation.IsDNS1123Subdomain(s.Group)) != 0 || s.Group == "argoproj.io" || s.Group == "authentication.k8s.io" || s.Group == "authorization.k8s.io" {
				return fmt.Errorf("unsupported result API group")
			}
			if !strings.HasPrefix(s.DocumentPath, prefix) || !validResultPointer(s.DocumentPath) || s.ObservedAtPath != "" && (!strings.HasPrefix(s.ObservedAtPath, prefix) || !validResultPointer(s.ObservedAtPath)) || s.MaxAgeSeconds < 0 || s.MaxAgeSeconds > 86400 || s.MaxAgeSeconds != 0 && s.ObservedAtPath == "" {
				return fmt.Errorf("result paths/freshness must select public data")
			}
		}
		if _, err := resultSchema(r); err != nil {
			return err
		}
		if len(r.Mappings) > 64 || len(r.Views) == 0 || len(r.Views) > 16 {
			return fmt.Errorf("invalid result mappings or views")
		}
		fields := map[string]bool{}
		for _, m := range r.Mappings {
			if !validResultPointer(m.From) || len(validation.IsDNS1123Label(m.To)) != 0 || fields[m.To] {
				return fmt.Errorf("invalid result projection")
			}
			fields[m.To] = true
		}
		for _, v := range r.Views {
			if !slices.Contains([]string{"summary", "table", "tree", "text"}, v.Type) || !validResultPointer(v.Path) || !validResultPointer(v.ChildrenPath) || len(v.Title) > 256 || len(v.Columns) > 32 || v.Type != "text" && len(v.Columns) == 0 || v.Type == "tree" && v.ChildrenPath == "" {
				return fmt.Errorf("invalid result view")
			}
			for _, c := range v.Columns {
				if !validResultPointer(c.Path) || len(c.Title) > 256 || !slices.Contains([]string{"", "text", "link", "status"}, c.Format) {
					return fmt.Errorf("invalid result column")
				}
			}
		}
	}
	return nil
}

// resultDefinitions selects the exact compiled call, including a precise-call
// wrapper's output aliases. It does not search by title or use live definitions.
func resultDefinitions(defs *api.ExecutionDefinitions, node wfv1.NodeStatus) ([]api.ActionResultDefinition, error) {
	if defs == nil || defs.ActionBinding == nil {
		return nil, nil
	}
	for _, call := range defs.ActionBinding.Calls {
		wrapper := false
		match := call.Template != "" && node.TemplateName == call.Template
		if len(call.CallPath) > 0 {
			key := p.Digest(strings.Join(call.CallPath, "/"))[:24]
			wrapper = node.TemplateName == "call-"+key
			match = wrapper || node.TemplateName == "action-"+key
		}
		if !match {
			continue
		}
		def, ok := defs.Actions[call.ActionDefinition]
		if !ok {
			return nil, fmt.Errorf("frozen action is missing")
		}
		results := def.DeepCopy().Results
		if len(call.ResultBindings) > 0 {
			frozen, exists := defs.CallResults[resultCallKey(call)]
			if !exists {
				return nil, fmt.Errorf("frozen result binding is missing")
			}
			results = make([]api.ActionResultDefinition, len(frozen))
			for i := range frozen {
				results[i] = *frozen[i].DeepCopy()
			}
		}
		if wrapper {
			mapped := map[string]string{}
			for _, m := range call.OutputMappings {
				mapped[m.From] = m.To
			}
			name := func(s string) string {
				if x := mapped[s]; x != "" {
					return x
				}
				return s
			}
			inputNames := map[string]string{}
			for _, m := range call.InputMappings {
				inputNames[m.To] = m.From
			}
			for i := range results {
				if resource := results[i].Source.Resource; resource != nil {
					if v := inputNames[resource.NameInput]; v != "" {
						resource.NameInput = v
					}
					if v := inputNames[resource.UIDInput]; v != "" {
						resource.UIDInput = v
					}
				}
				s := &results[i].Source
				s.Parameter = name(s.Parameter)
				if s.Resource != nil {
					s.Resource.NameOutput = name(s.Resource.NameOutput)
					s.Resource.UIDOutput = name(s.Resource.UIDOutput)
				}
			}
		}
		return results, nil
	}
	return nil, nil
}

func resultOutput(node wfv1.NodeStatus, name string) (string, bool) {
	if node.Outputs == nil {
		return "", false
	}
	if name == "result" && node.Outputs.Result != nil {
		return *node.Outputs.Result, true
	}
	for _, o := range node.Outputs.Parameters {
		if o.Name == name && o.Value != nil {
			return o.Value.String(), true
		}
	}
	return "", false
}

func resultInput(node wfv1.NodeStatus, name string) (string, bool) {
	if node.Inputs != nil {
		for _, p := range node.Inputs.Parameters {
			if p.Name == name && p.Value != nil {
				return p.Value.String(), true
			}
		}
	}
	return "", false
}

func normalizeResult(r api.ActionResultDefinition, value any) (any, error) {
	if r.Source.Format == "json" {
		if raw, ok := value.(string); ok {
			if len(raw) > maxResultBytes || json.Unmarshal([]byte(raw), &value) != nil {
				return nil, fmt.Errorf("ResultFormatInvalid")
			}
		}
	} else if _, ok := value.(string); !ok {
		return nil, fmt.Errorf("ResultFormatInvalid")
	}
	if len(r.Mappings) > 0 {
		projected := map[string]any{}
		for _, mapping := range r.Mappings {
			field, found := p.Pointer(value, mapping.From)
			if !found {
				return nil, fmt.Errorf("ResultFieldMissing")
			}
			projected[mapping.To] = field
		}
		value = projected
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > maxResultBytes {
		return nil, fmt.Errorf("ResultTooLarge")
	}
	compiled, err := resultSchema(r)
	if err != nil {
		return nil, err
	}
	valid, err := compiled.Validate(gojsonschema.NewGoLoader(value))
	if err != nil || !valid.Valid() {
		return nil, fmt.Errorf("ResultSchemaMismatch")
	}
	return value, nil
}

func (s Store) readResult(ctx context.Context, wf *wfv1.Workflow, node wfv1.NodeStatus, r api.ActionResultDefinition) NodeResult {
	out := NodeResult{Name: r.Name, Title: r.Title, Views: r.Views, Scope: "ExecutionSnapshot"}
	fail := func(code string) NodeResult { out.Error = code; return out }
	var value any
	if r.Source.Parameter != "" {
		x, found := resultOutput(node, r.Source.Parameter)
		if !found {
			return fail("OutputPending")
		}
		value = x
	} else {
		out.Scope = "LiveObservation"
		source := r.Source.Resource
		name, ok := resultOutput(node, source.NameOutput)
		uid, hasUID := resultOutput(node, source.UIDOutput)
		if source.NameInput != "" {
			name, ok = resultInput(node, source.NameInput)
			uid, hasUID = resultInput(node, source.UIDInput)
		}
		if !ok || !hasUID {
			return fail("ResourceIdentityPending")
		}
		if uid == "" || len(validation.IsDNS1123Subdomain(name)) != 0 {
			return fail("ResourceIdentityInvalid")
		}
		obj, err := s.Client.Resource(schema.GroupVersionResource{Group: source.Group, Version: source.Version, Resource: source.Resource}).Namespace(s.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
				return fail("AccessDenied")
			}
			if apierrors.IsNotFound(err) {
				return fail("ResourceMissing")
			}
			return fail("ObservationUnavailable")
		}
		if string(obj.GetUID()) != uid || obj.GetDeletionTimestamp() != nil {
			return fail("ResourceIdentityChanged")
		}
		owned := false
		for _, owner := range obj.GetOwnerReferences() {
			if owner.Kind == "Workflow" && owner.APIVersion == "argoproj.io/v1alpha1" && string(owner.UID) == string(wf.UID) {
				owned = true
			}
		}
		if !owned {
			return fail("ResourceOwnershipUnverified")
		}
		value, ok = p.Pointer(obj.Object, source.DocumentPath)
		if !ok {
			return fail("ObservationPending")
		}
		if source.ObservedAtPath != "" {
			timestamp, found := p.Pointer(obj.Object, source.ObservedAtPath)
			out.ObservedAt, _ = timestamp.(string)
			t, err := time.Parse(time.RFC3339Nano, out.ObservedAt)
			if !found || err != nil || t.After(time.Now().Add(time.Minute)) {
				return fail("ObservationTimestampInvalid")
			}
			out.Stale = source.MaxAgeSeconds > 0 && time.Since(t) > time.Duration(source.MaxAgeSeconds)*time.Second
		}
	}
	value, err := normalizeResult(r, value)
	if err != nil {
		return fail(err.Error())
	}
	out.Value, out.Available = value, true
	return out
}

// resultOrigin follows retained completed nodes across partial retries. A new
// receipt's definitions must not reinterpret an unrelated successful attempt.
// Every hop is a caller-authorized read of the exact same Workflow identity.
func (s Store) resultOrigin(ctx context.Context, wf *wfv1.Workflow, nodeID string) (*wfv1.Workflow, error) {
	seen := map[string]bool{}
	for depth := 0; depth < 32; depth++ {
		node, exists := wf.Status.Nodes[nodeID]
		name := wf.Annotations[p.Group+"/retry"]
		if !exists || node.Phase != wfv1.NodeSucceeded || name == "" {
			return wf, nil
		}
		if seen[name] {
			return nil, fmt.Errorf("result history cycle")
		}
		seen[name] = true
		obj, err := s.get(ctx, "parameterretries", name)
		if err != nil {
			return nil, err
		}
		var receipt api.ParameterRetrySpec
		if err = decode(obj, &receipt); err != nil {
			return nil, err
		}
		if receipt.Workflow != wf.Name || receipt.WorkflowUID != string(wf.UID) || receipt.Revision != wf.Annotations[p.Group+"/revision"] {
			return nil, fmt.Errorf("result history identity changed")
		}
		old := receipt.PreviousWorkflow
		if old == nil {
			return wf, nil
		}
		if old.Name != wf.Name || old.Namespace != wf.Namespace || old.UID != wf.UID {
			return nil, fmt.Errorf("result history belongs to another workflow")
		}
		previous, exists := old.Status.Nodes[nodeID]
		if !exists || p.ResultFingerprint(previous) != p.ResultFingerprint(node) {
			return wf, nil
		}
		wf = old
	}
	return nil, fmt.Errorf("result history exceeds the supported depth")
}

func (h Handler) nodeResults(ctx context.Context, s Store, workflow, uid, nodeID, history string) (*NodeResults, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	wf, err := h.Workflows.GetWorkflow(ctx, &workflowapi.WorkflowGetRequest{Namespace: s.Namespace, Name: workflow})
	if err != nil {
		return nil, err
	}
	if uid == "" || uid != string(wf.UID) {
		return nil, fmt.Errorf("workflow identity changed")
	}
	if history != "" {
		receipt, e := s.get(ctx, "parameterretries", history)
		if e != nil {
			return nil, e
		}
		var saved api.ParameterRetrySpec
		if e = decode(receipt, &saved); e != nil {
			return nil, e
		}
		if saved.Workflow != workflow || saved.WorkflowUID != uid || saved.PreviousWorkflow == nil || string(saved.PreviousWorkflow.UID) != uid || saved.PreviousWorkflow.Name != workflow || saved.PreviousWorkflow.Namespace != s.Namespace {
			return nil, fmt.Errorf("history belongs to another workflow")
		}
		wf = saved.PreviousWorkflow
	}
	wf, err = s.resultOrigin(ctx, wf, nodeID)
	if err != nil {
		return nil, err
	}
	node, ok := wf.Status.Nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node does not belong to this execution")
	}
	defs, err := s.executionDefinitions(ctx, wf)
	if err != nil {
		return nil, err
	}
	results, err := resultDefinitions(defs, node)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		template := definitionTemplate(&wf.Spec, node.TemplateName)
		if template == nil {
			return nil, fmt.Errorf("frozen result implementation is missing")
		}
		if err = validateActionResults(results, template); err != nil {
			return nil, err
		}
	}
	out := &NodeResults{Workflow: workflow, UID: uid, NodeID: nodeID, History: history, Invocation: p.Digest([]any{uid, nodeID, node.StartedAt, node.FinishedAt, node.Outputs}), ContractDigest: p.Digest(results), Items: []NodeResult{}}
	// GET performs only authorized reads. Neither a query failure nor a changed
	// health report may mutate native node state or dispatch a new attempt.
	for _, result := range results {
		out.Items = append(out.Items, s.readResult(ctx, wf, node, result))
	}
	return out, nil
}
