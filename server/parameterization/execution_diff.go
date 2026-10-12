package parameterization

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type ExecutionDiffRequest struct {
	Source   SourceExecution `json:"source"`
	Revision string          `json:"revision"`
	Limit    int             `json:"limit,omitempty"`
	Continue string          `json:"continue,omitempty"`
	Snapshot string          `json:"snapshot,omitempty"`
}
type ExecutionDiffLayer struct {
	Name      string     `json:"name"`
	Available bool       `json:"available"`
	Notice    string     `json:"notice,omitempty"`
	Changes   []p.Change `json:"changes"`
	Total     int        `json:"total"`
}
type ExecutionComparison struct {
	Source             SourceExecution              `json:"source"`
	ResourceVersion    string                       `json:"resourceVersion"`
	ExecutionDigest    string                       `json:"executionDigest"`
	Parameters         any                          `json:"parameters"`
	Layers             []ExecutionDiffLayer         `json:"layers"`
	DefinitionVersions map[string]map[string]string `json:"definitionVersions"`
	Notices            []string                     `json:"notices"`
	Pagination         map[string]any               `json:"pagination,omitempty"`
}

// executionDefinitions reads only immutable receipts, never a latest definition.
// Older receipts can lack snapshots; absence is reported instead of fabricated.
func (s Store) executionDefinitions(ctx context.Context, wf *wfv1.Workflow) (*parameterapi.ExecutionDefinitions, error) {
	if name := wf.Annotations[p.Group+"/retry"]; name != "" {
		obj, err := s.get(ctx, "parameterretries", name)
		if err != nil {
			return nil, err
		}
		var receipt parameterapi.ParameterRetrySpec
		if err = decode(obj, &receipt); err != nil {
			return nil, err
		}
		if receipt.Workflow != wf.Name || receipt.WorkflowUID != string(wf.UID) || receipt.Revision != wf.Annotations[p.Group+"/revision"] {
			return nil, fmt.Errorf("retry snapshot does not match current execution")
		}
		return receipt.Definitions, nil
	}
	name := wf.Annotations[p.Group+"/submission"]
	if name == "" {
		return nil, nil
	}
	obj, err := s.get(ctx, "workflowsubmissions", name)
	if err != nil {
		return nil, err
	}
	var receipt parameterapi.WorkflowSubmissionSpec
	if err = decode(obj, &receipt); err != nil {
		return nil, err
	}
	if receipt.Workflow.Name != wf.Name || receipt.Workflow.Namespace != wf.Namespace || receipt.Revision != wf.Annotations[p.Group+"/revision"] {
		return nil, fmt.Errorf("submission snapshot does not match current execution")
	}
	return receipt.Definitions, nil
}

func (h Handler) executionDiff(ctx context.Context, s Store, req ExecutionDiffRequest) (*ExecutionComparison, error) {
	source, err := h.editContext(ctx, s, req.Source)
	if err != nil {
		return nil, err
	}
	wf, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(s.Namespace).Get(ctx, req.Source.Workflow, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	stale := func() error {
		return apierrors.NewConflict(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, wf.Name, fmt.Errorf("execution inputs changed; reload comparison"))
	}
	if string(wf.UID) != req.Source.UID || wf.Annotations[p.Group+"/revision"] != source.Revision.GetName() {
		return nil, stale()
	}
	next, err := s.get(ctx, "parameterrevisions", req.Revision)
	if err != nil {
		return nil, err
	}
	var revision Revision
	if err = decode(next, &revision); err != nil {
		return nil, err
	}
	if revision.Set != source.Set.GetName() {
		return nil, fmt.Errorf("comparison revision belongs to another parameter set")
	}
	// This also rejects malformed sensitive values imported directly into CRs.
	comparison, err := s.compare(ctx, source.Revision.GetName(), req.Revision)
	if err != nil {
		return nil, err
	}
	previous, err := s.executionDefinitions(ctx, wf)
	if err != nil {
		return nil, err
	}
	preview, err := s.Preview(ctx, source.Binding, req.Revision)
	if err != nil {
		return nil, err
	}
	if len(preview.Errors) != 0 {
		return nil, fmt.Errorf("target revision is not valid for execution comparison")
	}
	result := &ExecutionComparison{Source: req.Source, ResourceVersion: wf.ResourceVersion, ExecutionDigest: preview.ExecutionDigest, Parameters: comparison, Layers: []ExecutionDiffLayer{}, DefinitionVersions: map[string]map[string]string{"target": preview.DefinitionVersions}, Notices: []string{
		"The left side is the saved execution; the right side is a read-only preview using current definitions. Comparing does not retry or submit a Workflow.",
		"Compiled inputs retain native expressions. Values depending on future task outputs are resolved by Argo during execution, not predicted by this comparison.",
	}}
	add := func(name string, left, right any, available bool) {
		layer := ExecutionDiffLayer{Name: name, Available: available, Changes: []p.Change{}}
		if available {
			layer.Changes = p.Diff(p.Clone(left), p.Clone(right), nil)
		} else {
			layer.Notice = "Historical definition content was not recorded by this receipt version. Current definitions are not used as its replacement."
		}
		result.Layers = append(result.Layers, layer)
	}
	if previous == nil {
		for _, name := range []string{"Parameter binding", "Action binding", "Form", "Action definitions", "Result bindings"} {
			add(name, nil, nil, false)
		}
	} else {
		result.DefinitionVersions["source"] = previous.Versions
		add("Parameter binding", previous.ParameterBinding, preview.Definitions.ParameterBinding, true)
		add("Action binding", previous.ActionBinding, preview.Definitions.ActionBinding, true)
		add("Form", previous.Form, preview.Definitions.Form, true)
		add("Action definitions", actionIdentities(previous), actionIdentities(preview.Definitions), true)
		add("Result bindings", previous.CallResults, preview.Definitions.CallResults, true)
	}
	oldContracts, err := compiledContracts(wf)
	if err != nil {
		return nil, err
	}
	newContracts, err := compiledContracts(preview.Workflow)
	if err != nil {
		return nil, err
	}
	add("Action contracts", oldContracts, newContracts, true)
	add("Implementations and workflow structure", compiledImplementations(wf), compiledImplementations(preview.Workflow), true)
	add("Compiled inputs", compiledInputs(wf), compiledInputs(preview.Workflow), true)
	// A concurrent native status update is harmless. A retry or input edit is not.
	current, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(s.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if current.UID != wf.UID || !sameParameterIdentity(current, wf) || current.Annotations[p.Group+"/retry"] != wf.Annotations[p.Group+"/retry"] || p.Digest(contractSpec(current)) != p.Digest(contractSpec(wf)) {
		return nil, stale()
	}
	return result, nil
}

func actionIdentities(definitions *parameterapi.ExecutionDefinitions) map[string]string {
	result := map[string]string{}
	for name, action := range definitions.Actions {
		result[name] = p.Digest(action)
	}
	return result
}
func compiledContracts(wf *wfv1.Workflow) (map[string]any, error) {
	result := map[string]any{}
	for _, template := range wf.Spec.Templates {
		contracts := map[string]any{}
		for _, item := range []struct{ name, key string }{{"input", p.InputContractAnnotation}, {"output", p.OutputContractAnnotation}} {
			encoded := template.Metadata.Annotations[item.key]
			if encoded == "" {
				continue
			}
			raw, err := base64.StdEncoding.DecodeString(encoded)
			var value any
			if err != nil || json.Unmarshal(raw, &value) != nil {
				return nil, fmt.Errorf("invalid saved action contract")
			}
			contracts[item.name] = value
		}
		if len(contracts) > 0 {
			result[template.Name] = contracts
		}
	}
	return result, nil
}
func compiledImplementations(wf *wfv1.Workflow) any {
	spec := contractSpec(wf)
	spec.Arguments = wfv1.Arguments{}
	for i := range spec.Templates {
		delete(spec.Templates[i].Metadata.Annotations, p.InputContractAnnotation)
		delete(spec.Templates[i].Metadata.Annotations, p.OutputContractAnnotation)
	}
	return spec
}
func compiledInputs(wf *wfv1.Workflow) any {
	templates := map[string]any{}
	for _, template := range wf.Spec.Templates {
		calls := map[string]any{}
		if template.DAG != nil {
			for _, task := range template.DAG.Tasks {
				calls[task.Name] = task.Arguments
			}
		}
		for _, group := range template.Steps {
			for _, step := range group.Steps {
				calls[step.Name] = step.Arguments
			}
		}
		templates[template.Name] = map[string]any{"inputs": template.Inputs, "calls": calls}
	}
	return map[string]any{"workflow": wf.Spec.Arguments, "templates": templates}
}
