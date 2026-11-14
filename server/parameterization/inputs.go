package parameterization

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type NodeInputBinding = parameterapi.NodeInputBinding
type FrozenInputForm struct {
	Model    p.Model        `json:"model"`
	Form     map[string]any `json:"form,omitempty"`
	Mappings []p.Mapping    `json:"mappings"`
}

// InputContext identifies a single waiting node's immutable input contract.
type InputContext struct {
	Workflow       string `json:"workflow"`
	UID            string `json:"uid"`
	NodeID         string `json:"nodeId"`
	ContractDigest string `json:"contractDigest"`
}
type NodeInputRequest struct {
	UID             string `json:"uid"`
	ResourceVersion string `json:"resourceVersion"`
	NodeID          string `json:"nodeId"`
	RequestID       string `json:"requestId"`
	Values          any    `json:"values"`
}
type PendingInput struct {
	NodeID         string          `json:"nodeId"`
	Name           string          `json:"name"`
	Contract       FrozenInputForm `json:"contract"`
	ContractDigest string          `json:"contractDigest"`
}
type PendingInputs struct {
	UID                  string           `json:"uid"`
	ResourceVersion      string           `json:"resourceVersion"`
	Items                []PendingInput   `json:"items"`
	Submitted            []map[string]any `json:"submitted"`
	EncryptedSecretInput bool             `json:"encryptedSecretInput"`
}

func (s Store) freezeInputForms(ctx context.Context, bindings []NodeInputBinding, wf *wfv1.Workflow, versions map[string]string) (map[string]FrozenInputForm, error) {
	result := map[string]FrozenInputForm{}
	for _, binding := range bindings {
		if _, exists := result[binding.Template]; exists {
			return nil, fmt.Errorf("duplicate input binding")
		}
		var template *wfv1.Template
		for i := range wf.Spec.Templates {
			if wf.Spec.Templates[i].Name == binding.Template {
				template = &wf.Spec.Templates[i]
			}
		}
		if template == nil || template.Suspend == nil {
			return nil, fmt.Errorf("input form must bind a native suspend template")
		}
		model, err := s.get(ctx, "parametertemplates", binding.ParameterTemplate)
		if err != nil {
			return nil, err
		}
		var contract FrozenInputForm
		if err = decode(model, &contract.Model); err != nil {
			return nil, err
		}
		if len(p.ValidateModelSchema(contract.Model)) > 0 {
			return nil, fmt.Errorf("input form has unsupported schema")
		}
		var supplied []wfv1.Parameter
		for _, out := range template.Outputs.Parameters {
			if out.ValueFrom != nil && out.ValueFrom.Supplied != nil {
				supplied = append(supplied, out)
			}
		}
		if err = checkStaticMappings(contract.Model, binding.Mappings, supplied); err != nil {
			return nil, err
		}
		contract.Mappings = binding.Mappings
		seen := map[string]bool{}
		for _, mapping := range binding.Mappings {
			if seen[mapping.Parameter] {
				return nil, fmt.Errorf("duplicate suspend output mapping")
			}
			seen[mapping.Parameter] = true
			found := false
			for _, out := range template.Outputs.Parameters {
				if out.Name == mapping.Parameter && out.ValueFrom != nil && out.ValueFrom.Supplied != nil {
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("input mapping must target a supplied native suspend output")
			}
		}
		for _, out := range template.Outputs.Parameters {
			if out.ValueFrom != nil && out.ValueFrom.Supplied != nil && !seen[out.Name] {
				return nil, fmt.Errorf("supplied output has no input mapping")
			}
		}
		versions["inputModel/"+binding.Template] = definitionContentVersion(model)
		if binding.Form != "" {
			form, err := s.get(ctx, "workflowforms", binding.Form)
			if err != nil {
				return nil, err
			}
			if err = decode(form, &contract.Form); err != nil {
				return nil, err
			}
			var declared parameterapi.WorkflowFormSpec
			if err = decode(form, &declared); err != nil {
				return nil, err
			}
			if err = validateForm(declared, contract.Model, binding.ParameterTemplate); err != nil {
				return nil, err
			}
			versions["inputForm/"+binding.Template] = definitionContentVersion(form)
		}
		result[binding.Template] = contract
	}
	return result, nil
}

func (s Store) inputForms(ctx context.Context, wf *wfv1.Workflow) (map[string]FrozenInputForm, error) {
	name := wf.Annotations[p.Group+"/submission"]
	if name == "" {
		return nil, fmt.Errorf("workflow has no parameter submission")
	}
	record, err := s.get(ctx, "workflowsubmissions", name)
	if err != nil {
		return nil, err
	}
	var snapshot struct {
		Workflow *wfv1.Workflow             `json:"workflow"`
		Forms    map[string]FrozenInputForm `json:"inputForms"`
	}
	if err = decode(record, &snapshot); err != nil {
		return nil, err
	}
	if snapshot.Workflow == nil || snapshot.Workflow.Name != wf.Name || snapshot.Workflow.Namespace != s.Namespace {
		return nil, fmt.Errorf("workflow submission identity mismatch")
	}
	return snapshot.Forms, nil
}
func (h Handler) pendingInputs(ctx context.Context, s Store, name string) (*PendingInputs, error) {
	wf, err := h.Workflows.GetWorkflow(ctx, &workflowapi.WorkflowGetRequest{Namespace: s.Namespace, Name: name})
	if err != nil {
		return nil, err
	}
	forms, err := s.inputForms(ctx, wf)
	if err != nil {
		return nil, err
	}
	result := &PendingInputs{UID: string(wf.UID), ResourceVersion: wf.ResourceVersion, Items: []PendingInput{}, EncryptedSecretInput: h.SecretWriter != nil}
	for id, node := range wf.Status.Nodes {
		if contract, ok := forms[node.TemplateName]; ok && node.IsActiveSuspendNode() {
			result.Items = append(result.Items, PendingInput{NodeID: id, Name: node.DisplayName, Contract: contract, ContractDigest: p.Digest(contract)})
		}
	}
	for key, name := range wf.Annotations {
		if !strings.HasPrefix(key, p.Group+"/input-") {
			continue
		}
		record, err := s.get(ctx, "parameternodeinputs", name)
		if err != nil {
			return nil, err
		}
		var data map[string]any
		if err = decode(record, &data); err != nil {
			return nil, err
		}
		if data["workflowUID"] != string(wf.UID) {
			return nil, fmt.Errorf("input receipt workflow identity mismatch")
		}
		result.Submitted = append(result.Submitted, data)
	}
	sort.Slice(result.Submitted, func(i, j int) bool {
		return fmt.Sprint(result.Submitted[i]["nodeId"]) < fmt.Sprint(result.Submitted[j]["nodeId"])
	})
	sort.Slice(result.Items, func(i, j int) bool { return result.Items[i].NodeID < result.Items[j].NodeID })
	return result, nil
}
func (h Handler) submitInput(ctx context.Context, s Store, name string, req NodeInputRequest) (*wfv1.Workflow, error) {
	if req.UID == "" || req.ResourceVersion == "" || req.NodeID == "" || req.RequestID == "" || len(req.RequestID) > 128 {
		return nil, fmt.Errorf("workflow identity, resourceVersion, nodeId and requestId required")
	}
	wf, err := h.Workflows.GetWorkflow(ctx, &workflowapi.WorkflowGetRequest{Namespace: s.Namespace, Name: name})
	if err != nil {
		return nil, err
	}
	if string(wf.UID) != req.UID {
		return nil, fmt.Errorf("workflow identity changed")
	}
	receiptName := "input-" + p.Digest([]string{req.UID, req.NodeID, req.RequestID})[:40]
	annotation := p.Group + "/input-" + p.Digest(req.NodeID)[:16]
	receipt, err := s.get(ctx, "parameternodeinputs", receiptName)
	if err == nil {
		digest, _, _ := unstructured.NestedString(receipt.Object, "spec", "requestDigest")
		if digest != p.Digest(req) {
			return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: p.Group, Resource: "parameternodeinputs"}, receiptName)
		}
		if wf.Annotations[annotation] == receiptName {
			return wf, nil
		}
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}
	if wf.ResourceVersion != req.ResourceVersion {
		return nil, apierrors.NewConflict(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, name, fmt.Errorf("workflow changed; refresh pending inputs"))
	}
	node, ok := wf.Status.Nodes[req.NodeID]
	if !ok || !node.IsActiveSuspendNode() {
		return nil, fmt.Errorf("target is not a waiting native suspend node")
	}
	forms, err := s.inputForms(ctx, wf)
	if err != nil {
		return nil, err
	}
	contract, ok := forms[node.TemplateName]
	if !ok {
		return nil, fmt.Errorf("node has no frozen input form")
	}
	values, sources, validationErrors := p.Resolve(contract.Model, req.Values)
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("node inputs fail schema validation; no input was applied")
	}
	optionVersions, err := s.validateInputOptions(ctx, &contract, values)
	if err != nil {
		return nil, err
	}
	secretVersions := map[string]string{}
	if receipt != nil {
		var frozen parameterapi.ParameterNodeInputSpec
		if err = decode(receipt, &frozen); err != nil {
			return nil, err
		}
		values = p.Clone(frozen.Values)
		secretVersions = frozen.SecretVersions
	} else if err = s.resolveLatestSecrets(ctx, contract.Model, values, secretVersions); err != nil {
		return nil, err
	}

	outputs, err := p.Arguments(contract.Model, values, contract.Mappings)
	if err != nil {
		return nil, err
	}
	if err = s.authorizeMappedSecrets(ctx, contract.Model, values, contract.Mappings, secretVersions); err != nil {
		return nil, err
	}
	if receipt != nil {
		var existing parameterapi.ParameterNodeInputSpec
		if err = decode(receipt, &existing); err != nil {
			return nil, err
		}
		if p.Digest(p.Clone(existing.Contract)) != p.Digest(p.Clone(contract)) {
			return nil, fmt.Errorf("input receipt contract mismatch")
		}
		if len(optionVersions) > 0 && p.Digest(existing.OptionVersions) != p.Digest(optionVersions) {
			return nil, apierrors.NewConflict(schema.GroupResource{Group: p.Group, Resource: "parameternodeinputs"}, receiptName, fmt.Errorf("option source changed before input commit; refresh and submit a new input request"))
		}
		if err = validatePinnedSecrets(ctx, s.Namespace, existing.SecretVersions); err != nil {
			return nil, err
		}
	}
	if receipt == nil {
		record := object("ParameterNodeInput", receiptName, s.Namespace, map[string]any{"requestDigest": p.Digest(req), "workflowUID": req.UID, "workflow": name, "nodeId": req.NodeID, "values": values, "sources": sources, "contract": contract, "secretVersions": secretVersions, "optionVersions": optionVersions, "actor": s.Actor, "submittedAt": time.Now().UTC().Format(time.RFC3339)})
		if _, err = s.resource("parameternodeinputs").Create(ctx, record, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return h.submitInput(ctx, s, name, req)
			}
			return nil, err
		}
	}
	encoded, _ := json.Marshal(outputs)
	return h.Workflows.SetWorkflow(p.WithInputProof(ctx, p.NativeInputProof{UID: req.UID, ResourceVersion: req.ResourceVersion, NodeID: req.NodeID, Receipt: receiptName}), &workflowapi.WorkflowSetRequest{Namespace: s.Namespace, Name: name, NodeFieldSelector: "id=" + req.NodeID, Phase: string(wfv1.NodeSucceeded), OutputParameters: string(encoded), Message: "validated parameter input received"})
}

// Resolve only the immutable submission, never the mutable catalog model.
func (h Handler) inputContract(ctx context.Context, s Store, req InputContext) (*FrozenInputForm, error) {
	if req.Workflow == "" || req.UID == "" || req.NodeID == "" || req.ContractDigest == "" {
		return nil, fmt.Errorf("workflow, UID, node and frozen contract digest required")
	}
	wf, err := h.Workflows.GetWorkflow(ctx, &workflowapi.WorkflowGetRequest{Namespace: s.Namespace, Name: req.Workflow})
	if err != nil {
		return nil, err
	}
	if string(wf.UID) != req.UID {
		return nil, fmt.Errorf("workflow identity changed")
	}
	node, ok := wf.Status.Nodes[req.NodeID]
	if !ok || !node.IsActiveSuspendNode() {
		return nil, fmt.Errorf("target is not a waiting native suspend node")
	}
	forms, err := s.inputForms(ctx, wf)
	if err != nil {
		return nil, err
	}
	contract, ok := forms[node.TemplateName]
	if !ok || p.Digest(contract) != req.ContractDigest {
		return nil, fmt.Errorf("frozen input contract mismatch")
	}
	return &contract, nil
}
