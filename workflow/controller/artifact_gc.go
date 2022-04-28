package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/slice"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

const artifactGCComponent = "artifact-gc"

func (wfc *WorkflowController) garbageCollect(ctx context.Context, obj interface{}) error {
	un := obj.(*unstructured.Unstructured)
	strategies := map[wfv1.ArtifactGCStrategy]bool{}

	if phase, ok := un.GetLabels()[common.LabelKeyPhase]; ok && wfv1.WorkflowPhase(phase).Completed() {
		strategies[wfv1.ArtifactGCOnWorkflowCompletion] = true
	}
	if un.GetDeletionTimestamp() != nil {
		strategies[wfv1.ArtifactGCOnWorkflowCompletion] = true
		strategies[wfv1.ArtfactGCOnWorkflowDeletion] = true
	}

	if len(strategies) == 0 {
		return nil
	}

	pods, err := wfc.podInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, un.GetNamespace()+"/"+un.GetName())
	if err != nil {
		return err
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		return err
	}

	log := log.WithField("workflow", wf.Name).WithField("namespace", wf.Namespace)

	if err := wfc.hydrator.Hydrate(wf); err != nil {
		return err
	}

	updated := false

	for _, obj := range pods {
		pod := obj.(*corev1.Pod)
		nodeID := pod.Annotations[common.EnvVarNodeID]
		artifactName := pod.Annotations[common.AnnotationArtifactName]
		phase := pod.Status.Phase
		log.WithField("pod", pod.Name).
			WithField("nodeID", nodeID).
			WithField("artifactName", artifactName).
			WithField("phase", phase).
			WithField("message", pod.Status.Message).
			Info("artifact-gc pod")
		if pod.Labels[common.LabelKeyComponent] != artifactGCComponent {
			continue
		}

		if phase == corev1.PodSucceeded {
			n := wf.Status.Nodes[nodeID]
			for i, a := range n.Outputs.Artifacts {
				if a.Name == artifactName {
					if !a.Deleted {
						updated = true
					}
					a.Deleted = true
					n.Outputs.Artifacts[i] = a
				}
			}
			wf.Status.Nodes[n.ID] = n
		}
	}

	if updated {
		if err := wfc.hydrator.Dehydrate(wf); err != nil {
			return err
		}
		_, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, wf, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	for _, obj := range pods {
		pod := obj.(*corev1.Pod)
		phase := pod.Status.Phase
		log.WithField("pod", pod.Name).
			WithField("phase", phase).
			WithField("message", pod.Status.Message).
			Info("artifact-gc pod")
		if phase == corev1.PodSucceeded {
			wfc.queuePodForCleanup(wf.Namespace, pod.Name, deletePod)
		}
	}

	if err := wfc.deleteArtifacts(ctx, wf, strategies); err != nil {
		return err
	}

	if un.GetDeletionTimestamp() == nil {
		return nil
	}

	return wfc.removeArtifactGCFinalizer(ctx, un)
}

func (wfc *WorkflowController) removeArtifactGCFinalizer(ctx context.Context, wf metav1.Object) error {
	data, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": slice.RemoveString(wf.GetFinalizers(), common.FinalizerArtifactGC),
		},
	})

	_, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(wf.GetNamespace()).Patch(ctx, wf.GetName(), types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		return err
	}
	return nil
}

func (wfc *WorkflowController) deleteArtifacts(ctx context.Context, wf *wfv1.Workflow, strategies map[wfv1.ArtifactGCStrategy]bool) error {

	as := wf.SearchArtifacts(&wfv1.ArtifactSearchQuery{ArtifactGCStrategies: strategies, Deleted: pointer.BoolPtr(false)})

	log.WithField("strategies", strategies).
		WithField("namespace", wf.GetNamespace()).
		WithField("name", wf.GetName()).
		WithField("artifacts", len(as)).
		Info("artifact garbage collection")

	if len(as) == 0 {
		return nil
	}

	for _, a := range as {
		if err := wfc.deleteArtifact(ctx, wf, &a); err != nil {
			return err
		}
	}
	return nil
}

func (wfc *WorkflowController) deleteArtifact(ctx context.Context, wf *wfv1.Workflow, a *wfv1.ArtifactSearchResult) error {
	ar, err := wfc.artifactRepositories.Get(ctx, wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return err
	}
	if err := a.Relocate(ar.ToArtifactLocation()); err != nil {
		return err
	}

	log.WithField("name", a.Name).
		WithField("namespace", wf.Namespace).
		WithField("workflow", wf.Name).
		WithField("nodeID", a.NodeID).
		Info("deleting artifact")

	data, err := json.Marshal(a)
	if err != nil {
		return err
	}

	volumes, volumeMounts := createSecretVolumes(&wfv1.Template{Outputs: wfv1.Outputs{Artifacts: []wfv1.Artifact{a.Artifact}}})

	_, err = wfc.kubeclientset.CoreV1().Pods(wf.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%v", wf.Name, a.NodeID),
			Annotations: map[string]string{
				common.AnnotationKeyNodeID:    a.NodeID,
				common.AnnotationArtifactName: a.Name,
			},
			Labels: map[string]string{
				common.LabelKeyWorkflow:  wf.Name,
				common.LabelKeyComponent: artifactGCComponent,
				// TODO instance ID
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: corev1.PodSpec{
			Volumes: volumes,
			Containers: []corev1.Container{
				{
					Name:            "main",
					Image:           wfc.executorImage(),
					ImagePullPolicy: wfc.executorImagePullPolicy(),
					Args:            []string{"artifact", "delete", "--loglevel", getExecutorLogLevel()},
					Env: []corev1.EnvVar{
						{Name: common.EnvVarArtifact, Value: string(data)},
					},
					SecurityContext: &corev1.SecurityContext{
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						Privileged:               pointer.Bool(false),
						RunAsNonRoot:             pointer.Bool(true),
						RunAsUser:                pointer.Int64Ptr(8737),
						ReadOnlyRootFilesystem:   pointer.Bool(true),
						AllowPrivilegeEscalation: pointer.Bool(false),
					},
					VolumeMounts: volumeMounts,
				},
			},
			RestartPolicy:      corev1.RestartPolicyOnFailure,
			ServiceAccountName: wf.GetExecSpec().ServiceAccountName,
		},
	}, metav1.CreateOptions{})

	if err != nil && !apierr.IsAlreadyExists(err) {
		return err
	}
	return nil
}
