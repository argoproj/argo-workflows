package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// NewDispatcher returns the controller's side over the Kubernetes API.
func NewDispatcher(client versioned.Interface) Dispatcher {
	return &dispatcher{store: kubeStore{client: client}}
}

// WatchSources are the watch-backed parts of a Kubernetes Consumer; a
// worker gets both from NewWatches.
type WatchSources struct {
	// Candidates finds the worker's candidates. Without it, Candidates fails
	// with ErrNotImplemented.
	Candidates CandidateSource
	// Claims watches claims. Without it, a claim learns it was lost only when
	// a Progress or Complete fails, and never learns of an interrupt.
	Claims ClaimWatcher
}

// NewConsumer returns a worker's side over the Kubernetes API.
func NewConsumer(client versioned.Interface, worker Worker, sources WatchSources) Consumer {
	return &consumer{
		store:      kubeStore{client: client},
		worker:     worker,
		candidates: sources.Candidates,
		watcher:    sources.Claims,
	}
}

type kubeStore struct {
	client versioned.Interface
}

// storeErr maps the API errors the protocol branches on, keeping the original.
func storeErr(err error) error {
	switch {
	case err == nil:
		return nil
	case apierr.IsNotFound(err):
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	case apierr.IsAlreadyExists(err):
		return fmt.Errorf("%w: %w", errAlreadyExists, err)
	}
	return err
}

// deleteErr is storeErr for a delete with a UID precondition, which the API
// server rejects as a conflict when the object has another UID.
func deleteErr(err error) error {
	if apierr.IsConflict(err) {
		return fmt.Errorf("%w: %w", ErrStale, err)
	}
	return storeErr(err)
}

func uidPrecondition(uid types.UID) metav1.DeleteOptions {
	return metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}}
}

func (s kubeStore) createTaskSet(ctx context.Context, taskSet *wfv1.WorkflowTaskSet) (*wfv1.WorkflowTaskSet, error) {
	created, err := s.client.ArgoprojV1alpha1().WorkflowTaskSets(taskSet.Namespace).Create(ctx, taskSet, metav1.CreateOptions{})
	return created, storeErr(err)
}

func (s kubeStore) getTaskSet(ctx context.Context, namespace, name string) (*wfv1.WorkflowTaskSet, error) {
	taskSet, err := s.client.ArgoprojV1alpha1().WorkflowTaskSets(namespace).Get(ctx, name, metav1.GetOptions{})
	return taskSet, storeErr(err)
}

func (s kubeStore) deleteTaskSet(ctx context.Context, namespace, name string, uid types.UID) error {
	return deleteErr(s.client.ArgoprojV1alpha1().WorkflowTaskSets(namespace).Delete(ctx, name, uidPrecondition(uid)))
}

func (s kubeStore) createResult(ctx context.Context, result *wfv1.WorkflowTaskResult) (*wfv1.WorkflowTaskResult, error) {
	created, err := s.client.ArgoprojV1alpha1().WorkflowTaskResults(result.Namespace).Create(ctx, result, metav1.CreateOptions{})
	return created, storeErr(err)
}

func (s kubeStore) getResult(ctx context.Context, namespace, name string) (*wfv1.WorkflowTaskResult, error) {
	result, err := s.client.ArgoprojV1alpha1().WorkflowTaskResults(namespace).Get(ctx, name, metav1.GetOptions{})
	return result, storeErr(err)
}

func (s kubeStore) listResults(ctx context.Context, namespace string, selector labels.Selector) ([]wfv1.WorkflowTaskResult, error) {
	list, err := s.client.ArgoprojV1alpha1().WorkflowTaskResults(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, storeErr(err)
	}
	return list.Items, nil
}

func (s kubeStore) deleteResult(ctx context.Context, namespace, name string, uid types.UID) error {
	return deleteErr(s.client.ArgoprojV1alpha1().WorkflowTaskResults(namespace).Delete(ctx, name, uidPrecondition(uid)))
}

type jsonPatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

var jsonPointerEscaper = strings.NewReplacer("~", "~0", "/", "~1")

func labelPath(key string) string {
	return "/metadata/labels/" + jsonPointerEscaper.Replace(key)
}

// jsonPatch is the patch as JSON patch operations: tests first, so the API
// server applies nothing unless they all pass.
func (p resultPatch) jsonPatch() ([]byte, error) {
	var ops []jsonPatchOp
	if p.workerPodUID != "" {
		ops = append(ops, jsonPatchOp{Op: "test", Path: labelPath(common.LabelKeyWorkerPodUID), Value: string(p.workerPodUID)})
	}
	ops = append(ops, jsonPatchOp{Op: "test", Path: "/metadata/uid", Value: string(p.uid)})
	for _, key := range slices.Sorted(maps.Keys(p.labels)) {
		ops = append(ops, jsonPatchOp{Op: "add", Path: labelPath(key), Value: p.labels[key]})
	}
	if p.phase != nil {
		ops = append(ops, jsonPatchOp{Op: "add", Path: "/phase", Value: *p.phase})
	}
	if p.message != nil {
		ops = append(ops, jsonPatchOp{Op: "add", Path: "/message", Value: *p.message})
	}
	if p.outputs != nil {
		ops = append(ops, jsonPatchOp{Op: "add", Path: "/outputs", Value: p.outputs})
	}
	if p.progress != nil {
		ops = append(ops, jsonPatchOp{Op: "add", Path: "/progress", Value: *p.progress})
	}
	return json.Marshal(ops)
}

func (s kubeStore) patchResult(ctx context.Context, namespace, name string, patch resultPatch) error {
	data, err := patch.jsonPatch()
	if err != nil {
		return err
	}
	results := s.client.ArgoprojV1alpha1().WorkflowTaskResults(namespace)
	_, err = results.Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	if !apierr.IsInvalid(err) {
		return storeErr(err)
	}
	// The API server reports a failed test as Invalid, as it does an invalid
	// patched object; only the live result can tell them apart.
	live, getErr := results.Get(ctx, name, metav1.GetOptions{})
	switch {
	case apierr.IsNotFound(getErr):
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	case getErr == nil && !patch.owns(live):
		return fmt.Errorf("%w: %w", ErrStale, err)
	case getErr != nil:
		return errors.Join(err, getErr)
	}
	return err
}
