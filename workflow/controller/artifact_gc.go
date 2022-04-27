package controller

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/slice"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/resource"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func (wfc *WorkflowController) garbageCollect(ctx context.Context, obj interface{}) {
	un := obj.(*unstructured.Unstructured)
	var strategies map[wfv1.ArtifactGCStrategy]bool

	if phase, ok := un.GetLabels()[common.LabelKeyPhase]; ok && wfv1.WorkflowPhase(phase).Completed() {
		strategies[wfv1.ArtifactGCOnWorkflowCompletion] = true
	}
	if un.GetDeletionTimestamp() != nil {
		strategies[wfv1.ArtifactGCOnWorkflowCompletion] = true
		strategies[wfv1.ArtfactGCOnWorkflowDeletion] = true
	}

	if err := wfc.deleteArtifacts(ctx, un, strategies); err != nil {
		log.WithError(err).Error("failed to delete artifacts")
		return
	}

	if un.GetDeletionTimestamp() == nil {
		return
	}

	data, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": slice.RemoveString(un.GetFinalizers(), Finalizer),
		},
	})

	_, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(un.GetNamespace()).Patch(ctx, un.GetName(), types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		log.WithError(err).Error("failed to remove artifact GC finalizer")
	}
}

func (wfc *WorkflowController) deleteArtifacts(ctx context.Context, un *unstructured.Unstructured, strategies map[wfv1.ArtifactGCStrategy]bool) error {
	wf, _ := util.FromUnstructured(un)
	log.WithField("strategies", strategies).
		WithField("namespace", un.GetNamespace()).
		WithField("name", un.GetName()).
		Info("artifact garbage collection")

	for _, a := range wf.SearchArtifacts(&wfv1.ArtifactSearchQuery{ArtifactGCStrategies: strategies}) {
		if err := wfc.deleteArtifact(ctx, wf, &a); err != nil {
			return err
		}
	}
	return nil
}

func (wfc *WorkflowController) deleteArtifact(ctx context.Context, wf *wfv1.Workflow, a *wfv1.Artifact) error {
	ar, err := wfc.artifactRepositories.Get(ctx, wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return err
	}
	l := ar.ToArtifactLocation()
	if err := a.Relocate(l); err != nil {
		return err
	}
	key, _ := a.GetKey()
	log.WithField("name", a.Name).
		WithField("key", key).
		Info("deleting artifact")
	drv, err := artifacts.NewDriver(ctx, a, resource.New(wfc.kubeclientset, wf.Namespace))
	if err != nil {
		return err
	}
	return drv.Delete(a)
}

// Finalizer prevents workflows from being deleted until they have had their artifacts GCed.
const Finalizer = "workflows.argoproj.io/artifact-gc"
