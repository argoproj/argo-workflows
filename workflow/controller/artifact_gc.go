package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/env"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/slice"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
)

const artifactGCComponent = "artifact-gc"

// artifactGCEnabled is a feature flag to globally disabled artifact GC in case of emergency
var artifactGCEnabled, _ = env.GetBool("ARGO_ARTIFACT_GC_ENABLED", true)

func (woc *wfOperationCtx) garbageCollectArtifacts(ctx context.Context) error {

	if !artifactGCEnabled {
		return nil
	}

	if !slice.ContainsString(woc.wf.Finalizers, common.FinalizerArtifactGC) {
		return nil
	}

	strategies := map[wfv1.ArtifactGCStrategy]bool{}

	if woc.wf.Labels[common.LabelKeyCompleted] == "true" || woc.wf.DeletionTimestamp != nil {
		strategies[wfv1.ArtifactGCOnWorkflowCompletion] = true
	}
	if woc.wf.DeletionTimestamp != nil {
		strategies[wfv1.ArtifactGCOnWorkflowDeletion] = true
	}

	if len(strategies) == 0 {
		woc.log.Debug("artifact GC not currently needed")
		return nil
	}

	pods, err := woc.controller.podInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.GetNamespace()+"/"+woc.wf.GetName())
	if err != nil {
		return fmt.Errorf("failed to get pods from informer: %w", err)
	}

	// go through any GC pods that are already running and may have completed
	// todo: break this out into its own method?
	podCount := 0
	for _, obj := range pods {
		pod := obj.(*corev1.Pod)
		if pod.Labels[common.LabelKeyComponent] != artifactGCComponent {
			continue
		}
		podCount++
		nodeID := pod.Annotations[common.AnnotationKeyNodeID]
		artifactName := pod.Annotations[common.AnnotationArtifactName]
		phase := pod.Status.Phase
		log.WithField("pod", pod.Name).
			WithField("nodeID", nodeID).
			WithField("artifactName", artifactName).
			WithField("phase", phase).
			WithField("message", pod.Status.Message).
			Info("reconciling artifact-gc pod")

		switch phase {
		case corev1.PodSucceeded:
			n := woc.wf.Status.Nodes[nodeID]
			for i, a := range n.Outputs.Artifacts {
				if a.Name == artifactName {
					a.Deleted = true
					n.Outputs.Artifacts[i] = a
				}
			}
			woc.wf.Status.Nodes[n.ID] = n
			woc.updated = true
			woc.controller.queuePodForCleanup(woc.wf.Namespace, pod.Name, deletePod)
		case corev1.PodFailed:
			woc.wf.Status.Conditions.UpsertCondition(wfv1.Condition{
				Type:    wfv1.ConditionTypeArtifactGCError,
				Status:  metav1.ConditionTrue,
				Message: fmt.Sprintf("%s/%s: %s", nodeID, artifactName, pod.Status.Message),
			})
			woc.updated = true
		}
	}

	maxConcurrency, err := env.GetInt("ARGO_ARTIFACT_MAX_CONCURRENT_PODS", 8)
	if err != nil {
		return fmt.Errorf("failed to get artifact max concurrent pods env var: %w", err)
	}
	if podCount >= maxConcurrency {
		woc.log.WithField("maxConcurrency", maxConcurrency).Info("max artifact concurrent pods reached")
		return nil
	}

	// search for the Artifacts that are currently deletable
	artifacts := woc.execWf.SearchArtifacts(&wfv1.ArtifactSearchQuery{ArtifactGCStrategies: strategies, Deleted: pointer.BoolPtr(false)})
	fmt.Printf("deletethis: SearchArtifacts for what's deletable returned: %+v\n", artifacts)

	// create pods for deleting those artifacts, if they don't already exist
	if err := woc.createPodsToDeleteArtifacts(ctx, artifacts); err != nil {
		return fmt.Errorf("failed to create pods to delete artifacts: %w", err)
	}

	// check to see if everything's been deleted
	remaining := woc.execWf.SearchArtifacts(&wfv1.ArtifactSearchQuery{ArtifactGCStrategies: wfv1.AnyArtifactGCStrategy, Deleted: pointer.BoolPtr(false)})
	fmt.Printf("deletethis: SearchArtifacts for remaining returned: %+v\n", remaining)

	if len(remaining) == 0 {
		woc.log.Info("no remaining artifacts to GC, removing artifact GC finalizer")
		woc.wf.Finalizers = slice.RemoveString(woc.wf.Finalizers, common.FinalizerArtifactGC)
		woc.updated = true
	}
	return nil
}

func (woc *wfOperationCtx) createPodsToDeleteArtifacts(ctx context.Context, artifacts wfv1.ArtifactSearchResults) error {
	for _, a := range artifacts {
		if err := woc.createArtifactGCPod(ctx, &a); err != nil {
			return fmt.Errorf("failed to delete artifact %q: %w", a.Name, err)
		}
	}
	return nil
}

func (woc *wfOperationCtx) createArtifactGCPod(ctx context.Context, a *wfv1.ArtifactSearchResult) error {
	h := fnv.New32()
	if _, err := h.Write([]byte(a.NodeID)); err != nil {
		return fmt.Errorf("failed to update hash with node ID: %w", err)
	}
	if _, err := h.Write([]byte(a.Name)); err != nil {
		return fmt.Errorf("failed to update hash with artifact name: %w", err)
	}

	podName := fmt.Sprintf("%s-agc-%x", woc.wf.Name, h.Sum32())

	// first make sure it doesn't already exist
	_, exists, err := woc.controller.podInformer.GetIndexer().GetByKey(woc.wf.Namespace + "/" + podName)
	if err != nil {
		return fmt.Errorf("failed to get pod by key: %w", err)
	}
	fmt.Printf("deletethis: checking if GC pod of name %s exists: %t\n", podName, exists)
	if exists {
		return nil
	}
	ar, err := woc.controller.artifactRepositories.Get(ctx, woc.wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return fmt.Errorf("failed to get artifact repository: %w", err)
	}
	if err := a.Relocate(ar.ToArtifactLocation()); err != nil {
		return fmt.Errorf("failed to relocate artifact: %w", err)
	}

	woc.log.
		WithField("nodeID", a.NodeID).
		WithField("artifactName", a.Name).
		Info("create pod to delete artifact")

	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshall artifact: %w", err)
	}

	volumes, volumeMounts := createSecretVolumes(&wfv1.Template{Outputs: wfv1.Outputs{Artifacts: []wfv1.Artifact{a.Artifact}}})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name,
				common.LabelKeyComponent: artifactGCComponent,
				common.LabelKeyCompleted: "false",
			},
			Annotations: map[string]string{
				common.AnnotationKeyNodeID:    a.NodeID,
				common.AnnotationArtifactName: a.Name,
			},
			OwnerReferences: []metav1.OwnerReference{ // make sure we get deleted with the workflow
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: corev1.PodSpec{
			Volumes: volumes,
			Containers: []corev1.Container{
				{
					Name:            common.MainContainerName,
					Image:           woc.controller.executorImage(),
					ImagePullPolicy: woc.controller.executorImagePullPolicy(),
					Args:            []string{"artifact", "delete", "--loglevel", getExecutorLogLevel()},
					Env: []corev1.EnvVar{
						{Name: common.EnvVarArtifact, Value: string(data)},
					},
					// if this pod is breached by an attacker we:
					// * prevent installation of any new packages
					// * modification of the file-system
					SecurityContext: &corev1.SecurityContext{
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						Privileged:               pointer.Bool(false),
						RunAsNonRoot:             pointer.Bool(true),
						RunAsUser:                pointer.Int64Ptr(8737),
						ReadOnlyRootFilesystem:   pointer.Bool(true),
						AllowPrivilegeEscalation: pointer.Bool(false),
					},
					// if this pod is breached by an attacker these limits prevent excessive CPU and memory usage
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("100m"), //todo: should these values be in the Controller config?
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
			// if this pod is breached by an attacker this prevents them making Kubernetes API requests
			AutomountServiceAccountToken: pointer.Bool(false),
			RestartPolicy:                corev1.RestartPolicyOnFailure,
			ServiceAccountName:           woc.execWf.Spec.ServiceAccountName,
		},
	}

	if v := woc.controller.Config.InstanceID; v != "" {
		pod.Labels[common.EnvVarInstanceID] = v
	}

	// we need to run using the same configuration to the template that created the artifact
	node := woc.wf.Status.Nodes[a.NodeID]
	tmpl := woc.execWf.GetTemplateByName(node.TemplateName)

	if v := tmpl.ServiceAccountName; v != "" {
		pod.Spec.ServiceAccountName = v
	}
	woc.addMetadata(pod, tmpl)

	_, err = woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	fmt.Printf("deletethis: attempted to create GC pod of name %s: err=%v\n ", pod.Name, err)

	if err != nil && !apierr.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create pod: %w", err)
	}
	return nil
}
