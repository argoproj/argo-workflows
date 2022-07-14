package controller

import (
	"context"
	"fmt"
	"hash/fnv"

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

	strategies := map[wfv1.ArtifactGCStrategy]struct{}{} // essentially a Set

	if woc.wf.Labels[common.LabelKeyCompleted] == "true" || woc.wf.DeletionTimestamp != nil {
		strategies[wfv1.ArtifactGCOnWorkflowCompletion] = struct{}{}
	}
	if woc.wf.DeletionTimestamp != nil {
		strategies[wfv1.ArtifactGCOnWorkflowDeletion] = struct{}{}
	}
	if woc.wf.Status.Successful() {
		strategies[wfv1.ArtifactGCOnWorkflowSuccess] = struct{}{}
	}
	if woc.wf.Status.Failed() {
		strategies[wfv1.ArtifactGCOnWorkflowFailure] = struct{}{}
	}

	if len(strategies) == 0 {
		woc.log.Debug("artifact GC not currently needed")
		return nil
	}

	for strategy, _ := range strategies {
		err := woc.processArtifactGCStrategy(strategy)
		if err != nil {
			return err
		}
	}
}

/*pods, err := woc.controller.podInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.GetNamespace()+"/"+woc.wf.GetName())
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
	}*/

func (woc *wfOperationCtx) processArtifactGCStrategy(strategy wfv1.ArtifactGCStrategy) {
	if woc.wf.Status.ArtifactGCStatus == nil {
		statusMap := make(wfv1.ArtifactGCStatus)
		woc.wf.Status.ArtifactGCStatus = &statusMap
	}
	podRan := false
	status, exists := (*woc.wf.Status.ArtifactGCStatus)[strategy]
	if !exists {
		(*woc.wf.Status.ArtifactGCStatus)[strategy] = wfv1.NodePending
	} else {
		podRan = (status == wfv1.NodeSucceeded || status == wfv1.NodeFailed)
	}

	if !podRan {

	}

	/*
		for strategy, _ := range strategies {

			for serviceAcct, templates := range templatesByServiceAcct {

				// get all of the Artifacts for this ServiceAccount and Strategy, so we can delete those in one Pod
				artifacts := make(wfv1.ArtifactSearchResults, 0)

				for _, template := range templates { // todo: consider optimizing this: it will walk through all nodes multiple times
					// search for the Artifacts that are currently deletable
					artifactsForTemplate := woc.execWf.SearchArtifacts(&wfv1.ArtifactSearchQuery{ArtifactGCStrategies: map[wfv1.ArtifactGCStrategy]bool{strategy: true}, TemplateName: template.Name, Deleted: pointer.BoolPtr(false)})
					fmt.Printf("deletethis: SearchArtifacts for what's deletable for strategy %v returned: %+v\n", strategy, artifacts)
					artifacts = append(artifacts, artifactsForTemplate...)
				}

				// create pods for deleting those artifacts, if they don't already exist
				if err := woc.createArtifactGCPod(ctx, strategy, serviceAcct, artifacts, templates); err != nil {
					return fmt.Errorf("failed to create pods to delete artifacts: %w", err)
				}

			}
		}

		// check to see if everything's been deleted
		remaining := woc.execWf.SearchArtifacts(&wfv1.ArtifactSearchQuery{ArtifactGCStrategies: wfv1.AnyArtifactGCStrategy, Deleted: pointer.BoolPtr(false)})
		fmt.Printf("deletethis: SearchArtifacts for remaining returned: %+v\n", remaining)

		if len(remaining) == 0 {
			woc.log.Info("no remaining artifacts to GC, removing artifact GC finalizer")
			woc.wf.Finalizers = slice.RemoveString(woc.wf.Finalizers, common.FinalizerArtifactGC)
			woc.updated = true
		}
		return nil*/
}

func (woc *wfOperationCtx) podName(workflowName string, strategy wfv1.ArtifactGCStrategy, serviceAcct string) (string, error) {
	h := fnv.New32()
	if _, err := h.Write([]byte(serviceAcct)); err != nil {
		return "", fmt.Errorf("failed to update hash with serviceAcct '%s': %w", serviceAcct, err)
	}
	return fmt.Sprintf("artgc-%s-%x", strategy.AbbreviatedName(), h.Sum32()), nil
}

func (woc *wfOperationCtx) getArtifactTaskSet(taskSetName string) (*wfv1.WorkflowTaskSet, error) {
	taskSet, exists, err := woc.controller.wfTaskSetInformer.Informer().GetIndexer().GetByKey(woc.wf.Namespace + "/" + taskSetName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return taskSet.(*wfv1.WorkflowTaskSet), nil
}

//	create a WorkflowTaskSet which contains a subset of our Workflow's spec (just the Templates for this Service Account)
func (woc *wfOperationCtx) createArtifactGCTaskSet(ctx context.Context, templates []*wfv1.Template, taskSetName string) error {

	// first make sure it doesn't already exist
	taskSet, err := woc.getArtifactTaskSet(taskSetName)
	if err != nil {
		return err
	}
	if taskSet != nil {
		woc.log.Debugf("Artifact GC Task Set %s already exists", taskSetName)
	} else {
		woc.log.Infof("Creating Artifact GC Task Set %s", taskSetName)

		nodesMap := woc.generateMapOfNodes(templates)

		taskSet := wfv1.WorkflowTaskSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       workflow.WorkflowTaskSetKind,
				APIVersion: workflow.APIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: woc.wf.Namespace,
				Name:      taskSetName,
				/*OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: woc.wf.APIVersion,
						Kind:       woc.wf.Kind,
						UID:        woc.wf.UID,
						Name:       woc.wf.Name,
					},
				},*/
			},
			Spec: wfv1.WorkflowTaskSetSpec{
				Tasks: nodesMap,
			},
		}

		_, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(woc.wf.Namespace).Create(ctx, &taskSet, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

//todo: determine if this is optimal wrt the SearchArtifacts query logic
func (woc *wfOperationCtx) generateMapOfNodes(templates []*wfv1.Template) map[string]wfv1.Template {

	// create map of templates by name so we can do easy look up of template names below
	templateMap := make(map[string]*wfv1.Template)
	for _, template := range templates {
		templateMap[template.Name] = template
	}

	nodeMap := make(map[string]wfv1.Template)

	for _, n := range woc.wf.Status.Nodes {
		template, found := templateMap[n.TemplateName]
		if found {
			nodeMap[n.ID] = *template
		}
	}
	return nodeMap
}

func (woc *wfOperationCtx) createArtifactGCPod(ctx context.Context, strategy wfv1.ArtifactGCStrategy, serviceAcct string, artifacts wfv1.ArtifactSearchResults, templates []*wfv1.Template) error {

	//	create a WorkflowTaskSet which contains a subset of our Workflow's spec (just the Templates for this Service Account), as well as a GC Pod with the following environment variables:
	//		- WorkflowTaskSet name
	//		- ArtifactGC Strategy
	//	need to set Ownership for both
	// Need to make sure that Template.ArchiveLocation grabs the ArtifactRepositoryRef! If this is what's being passed to the wait container, then that would need to have access to this so I assume it does?
	// although, this will nullify our whole "normalized format thing"

	// for each artifact
	//	get nodeID which enables us to get template
	//	get ServiceAccount from template

	podName, err := woc.podName(woc.wf.Name, strategy, serviceAcct)
	if err != nil {
		return err
	}

	// first make sure it doesn't already exist
	_, exists, err := woc.controller.podInformer.GetIndexer().GetByKey(woc.wf.Namespace + "/" + podName)
	if err != nil {
		return fmt.Errorf("failed to get pod by key: %w", err)
	}
	fmt.Printf("deletethis: checking if GC pod of name %s (for service account %s) exists: %t\n", podName, serviceAcct, exists)
	if exists {
		return nil
	}

	taskSetName := podName
	err = woc.createArtifactGCTaskSet(ctx, templates, taskSetName)
	if err != nil {
		return err
	}
	/*ar, err := woc.controller.artifactRepositories.Get(ctx, woc.wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return fmt.Errorf("failed to get artifact repository: %w", err)
	}
	if err := a.Relocate(ar.ToArtifactLocation()); err != nil {
		return fmt.Errorf("failed to relocate artifact: %w", err)
	}*/

	woc.log.
		WithField("strategy", strategy).
		WithField("serviceAcct", serviceAcct).
		Infof("create pod to delete artifacts: %s", podName)

	/*data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshall artifact: %w", err)
	}*/
	volumes, volumeMounts := createSecretVolumes(templates, false, true)

	//volumes, volumeMounts := createSecretVolumes(&wfv1.Template{Outputs: wfv1.Outputs{Artifacts: []wfv1.Artifact{a.Artifact}}})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name,
				common.LabelKeyComponent: artifactGCComponent,
				common.LabelKeyCompleted: "false",
			},
			/*Annotations: map[string]string{
				common.AnnotationKeyNodeID:    a.NodeID,
				common.AnnotationArtifactName: a.Name,
			},*/
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
						{Name: common.EnvVarArtifactGCStrategy, Value: string(strategy)},
						{Name: common.EnvVarArtifactGCTaskSet, Value: taskSetName},
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
			ServiceAccountName:           serviceAcct,
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
	woc.addMetadata(pod, tmpl) //todo: what's the need for this?

	_, err = woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	fmt.Printf("deletethis: attempted to create GC pod of name %s: err=%v\n ", pod.Name, err)

	if err != nil && !apierr.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create pod: %w", err)
	}
	return nil
}
