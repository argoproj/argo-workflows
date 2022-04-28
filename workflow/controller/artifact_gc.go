package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"

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

func (wfc *WorkflowController) garbageCollectArtifacts(ctx context.Context, obj interface{}) error {
	un := obj.(*unstructured.Unstructured)
	strategies := map[wfv1.ArtifactGCStrategy]bool{}

	if phase, ok := un.GetLabels()[common.LabelKeyPhase]; ok && wfv1.WorkflowPhase(phase).Completed() {
		strategies[wfv1.ArtifactGCOnWorkflowCompletion] = true
	}
	if un.GetDeletionTimestamp() != nil {
		strategies[wfv1.ArtifactGCOnWorkflowCompletion] = true
		strategies[wfv1.ArtifactGCOnWorkflowDeletion] = true
	}

	if len(strategies) == 0 {
		return nil
	}

	pods, err := wfc.podInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, un.GetNamespace()+"/"+un.GetName())
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		return fmt.Errorf("failed to unmarshall workflow: %w", err)
	}

	log := log.WithField("workflow", wf.Name).WithField("namespace", wf.Namespace)

	if err := wfc.hydrator.Hydrate(wf); err != nil {
		return fmt.Errorf("failed to hydrate workflow: %w", err)
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
			return fmt.Errorf("failed to dehydrate workflow: %w", err)
		}
		_, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, wf, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update workflow: %w", err)
		}
	}

	wfc.deleteSuccessfulArtifactGCPods(pods, log, wf)

	if err := wfc.createPodsToDeleteArtifacts(ctx, wf, strategies); err != nil {
		return fmt.Errorf("failed to delete artifacts: %w", err)
	}

	if un.GetDeletionTimestamp() == nil {
		return nil
	}

	if err := wfc.removeArtifactGCFinalizer(ctx, wf); err != nil {
		return fmt.Errorf("failed to remove finalizer: %w", err)
	}
	return nil
}

func (wfc *WorkflowController) deleteSuccessfulArtifactGCPods(pods []interface{}, log *log.Entry, wf *wfv1.Workflow) {
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
}

func (wfc *WorkflowController) removeArtifactGCFinalizer(ctx context.Context, wf metav1.Object) error {
	data, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": slice.RemoveString(wf.GetFinalizers(), common.FinalizerArtifactGC),
		},
	})

	_, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(wf.GetNamespace()).Patch(ctx, wf.GetName(), types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		return fmt.Errorf("failed to patch workflow finalizer: %w", err)
	}
	return nil
}

func (wfc *WorkflowController) createPodsToDeleteArtifacts(ctx context.Context, wf *wfv1.Workflow, strategies map[wfv1.ArtifactGCStrategy]bool) error {

	as := wf.SearchArtifacts(&wfv1.ArtifactSearchQuery{ArtifactGCStrategies: strategies, Deleted: pointer.BoolPtr(false)})

	log.WithField("strategies", strategies).
		WithField("namespace", wf.GetNamespace()).
		WithField("name", wf.GetName()).
		WithField("artifacts", len(as)).
		Info("artifact garbage collection")

	for _, a := range as {
		if err := wfc.createArtifactGCPod(ctx, wf, &a); err != nil {
			return fmt.Errorf("failed to delete artifact %q: %w", a.Name, err)
		}
	}
	return nil
}

func (wfc *WorkflowController) createArtifactGCPod(ctx context.Context, wf *wfv1.Workflow, a *wfv1.ArtifactSearchResult) error {
	h := fnv.New32()
	if _, err := h.Write([]byte(a.NodeID)); err != nil {
		return fmt.Errorf("failed to update hash with node ID: %w", err)
	}
	if _, err := h.Write([]byte(a.Name)); err != nil {
		return fmt.Errorf("failed to update hash with artifact name: %w", err)
	}

	podName := fmt.Sprintf("%s-%x", wf.Name, h.Sum32())

	_, exists, err := wfc.podInformer.GetIndexer().GetByKey(wf.Namespace + "/" + podName)
	if err != nil {
		return fmt.Errorf("failed to get pod by key: %w", err)
	}
	if exists {
		return nil
	}
	ar, err := wfc.artifactRepositories.Get(ctx, wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return fmt.Errorf("failed to get artifact repository: %w", err)
	}
	if err := a.Relocate(ar.ToArtifactLocation()); err != nil {
		return fmt.Errorf("failed to relocate artifact: %w", err)
	}

	log.WithField("name", a.Name).
		WithField("namespace", wf.Namespace).
		WithField("workflow", wf.Name).
		WithField("nodeID", a.NodeID).
		WithField("artifactName", a.Name).
		Info("create pod to delete artifact")

	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshall artifact: %w", err)
	}

	volumes, volumeMounts := createSecretVolumes(&wfv1.Template{Outputs: wfv1.Outputs{Artifacts: []wfv1.Artifact{a.Artifact}}})

	_, err = wfc.kubeclientset.CoreV1().Pods(wf.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  wf.Name,
				common.LabelKeyComponent: artifactGCComponent,
				// TODO instance ID
			},
			Annotations: map[string]string{
				common.AnnotationKeyNodeID:           a.NodeID,
				common.AnnotationArtifactName:        a.Name,
				common.AnnotationKeyDefaultContainer: common.MainContainerName,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: corev1.PodSpec{
			Volumes: volumes,
			Containers: []corev1.Container{
				{
					Name:            common.MainContainerName,
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
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("100m"),
							"memory": resource.MustParse("64Mi"),
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("50m"),
							"memory": resource.MustParse("32Mi"),
						},
					},
					VolumeMounts: volumeMounts,
				},
			},
			RestartPolicy:      corev1.RestartPolicyOnFailure,
			ServiceAccountName: wf.GetExecSpec().ServiceAccountName,
		},
	}, metav1.CreateOptions{})

	if err != nil && !apierr.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create pod: %w", err)
	}
	return nil
}
