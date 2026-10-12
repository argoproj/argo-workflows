package parameterization

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

type RetryRequest struct {
	Binding         string   `json:"binding,omitempty"`
	CallPaths       []string `json:"callPaths,omitempty"`
	Workflow        string   `json:"workflow"`
	Revision        string   `json:"revision,omitempty"`
	PlanDigest      string   `json:"planDigest,omitempty"`
	UID             string   `json:"uid"`
	ResourceVersion string   `json:"resourceVersion"`
	IdempotencyKey  string   `json:"idempotencyKey"`
}

func (h Handler) retry(ctx context.Context, s Store, req RetryRequest) (*wfv1.Workflow, error) {
	if req.IdempotencyKey == "" || len(req.IdempotencyKey) > 128 || req.UID == "" || req.ResourceVersion == "" {
		return nil, fmt.Errorf("retry needs workflow UID, resourceVersion and idempotency key")
	}
	wf, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(s.Namespace).Get(ctx, req.Workflow, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if string(wf.UID) != req.UID {
		return nil, fmt.Errorf("workflow identity changed")
	}
	name := "retry-" + p.Digest([]string{req.UID, req.IdempotencyKey})[:40]
	receipt, err := s.get(ctx, "parameterretries", name)
	if err == nil {
		digest, _, _ := unstructured.NestedString(receipt.Object, "spec", "requestDigest")
		if digest != p.Digest(req) {
			return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: p.Group, Resource: "parameterretries"}, name)
		}
		if wf.Annotations[p.Group+"/retry"] == name {
			return wf, nil
		}
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}
	if wf.ResourceVersion != req.ResourceVersion {
		return nil, apierrors.NewConflict(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, req.Workflow, fmt.Errorf("workflow changed; reload before retrying"))
	}
	if req.Revision != "" || len(req.CallPaths) > 0 {
		return h.retryChangedParameters(ctx, s, req)
	}
	revision := wf.Annotations[p.Group+"/revision"]
	if revision == "" {
		return nil, fmt.Errorf("this workflow has no parameter revision")
	}
	if wf.Status.Phase != wfv1.WorkflowFailed && wf.Status.Phase != wfv1.WorkflowError {
		return nil, fmt.Errorf("only failed or errored workflows can retry")
	}
	// Check present authorization to the original snapshot. Never resolve latest definitions.
	if _, err = s.get(ctx, "parameterrevisions", revision); err != nil {
		return nil, err
	}
	definitions, err := s.executionDefinitions(ctx, wf)
	if err != nil {
		return nil, err
	}
	// Reusing the execution snapshot never revives a revoked/replaced credential.
	// Read only the pinned identities, without resolving mutable definitions.
	if definitions != nil {
		if err = validatePinnedSecrets(ctx, s.Namespace, definitions.Versions); err != nil {
			return nil, err
		}
	}
	if err = s.validateInputSecrets(ctx, wf); err != nil {
		return nil, err
	}
	if receipt == nil {
		previous := wf.DeepCopy()
		if previous.Status.CompressedNodes != "" || previous.Status.OffloadNodeStatusVersion != "" {
			previous, err = h.Workflows.GetWorkflow(ctx, &workflowapi.WorkflowGetRequest{Namespace: s.Namespace, Name: req.Workflow})
			if err != nil {
				return nil, err
			}
			if previous.UID != wf.UID || previous.ResourceVersion != wf.ResourceVersion {
				return nil, fmt.Errorf("workflow changed while preserving retry results; reload before retrying")
			}
		}
		previous.APIVersion, previous.Kind = "argoproj.io/v1alpha1", "Workflow"
		receipt = object("ParameterRetry", name, s.Namespace, map[string]any{"requestDigest": p.Digest(req), "workflow": req.Workflow, "workflowUID": req.UID, "resourceVersion": req.ResourceVersion, "revision": revision, "actor": s.Actor, "definitions": definitions, "previousWorkflow": previous})
		if _, err = s.resource("parameterretries").Create(ctx, receipt, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return h.retry(ctx, s, req)
			}
			return nil, err
		}
	}
	return h.Workflows.RetryWorkflow(p.WithRetryProof(ctx, p.NativeRetryProof{UID: req.UID, ResourceVersion: req.ResourceVersion, Receipt: name}), &workflowapi.WorkflowRetryRequest{Namespace: s.Namespace, Name: req.Workflow})
}
