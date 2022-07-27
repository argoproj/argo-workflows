package controller

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"

	"golang.org/x/exp/maps"
	apiv1 "k8s.io/api/core/v1"
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

	// todo: don't want this here but want to see if it works
	/*repo, err := woc.controller.artifactRepositories.Get(ctx, woc.wf.Status.ArtifactRepositoryRef)
	if err != nil {
		woc.markWorkflowError(ctx, fmt.Errorf("failed to get artifact repository: %v", err))
		return err
	}
	woc.artifactRepository = repo*/

	if !artifactGCEnabled {
		return nil
	}

	if woc.wf.Status.ArtifactGCStatus == nil {
		woc.wf.Status.ArtifactGCStatus = &wfv1.ArtGCStatus{}
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
		woc.log.Debugf("processing Artifact GC Strategy %s", strategy)
		err := woc.processArtifactGCStrategy(ctx, strategy)
		if err != nil {
			return err
		}
	}

	err := woc.processArtifactGCCompletion(ctx)
	if err != nil {
		return err
	}
	return nil
}

// go through any GC pods that are already running and may have completed
func (woc *wfOperationCtx) processArtifactGCCompletion(ctx context.Context) error {
	// check if any previous Artifact GC Pods completed
	pods, err := woc.controller.podInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.GetNamespace()+"/"+woc.wf.GetName())
	if err != nil {
		return fmt.Errorf("failed to get pods from informer: %w", err)
	}

	anyPodSuccess := false
	for _, obj := range pods {
		pod := obj.(*corev1.Pod)
		if pod.Labels[common.LabelKeyComponent] != artifactGCComponent {
			continue
		}
		strategyStr, found := pod.Annotations[common.AnnotationKeyArtifactGCStrategy]
		if !found {
			return fmt.Errorf("Artifact GC Pod '%s' doesn't have annotation '%s'?", pod.Name, common.AnnotationKeyArtifactGCStrategy)
		}
		fmt.Printf("deletethis: processing pod %s for strategy %s\n", pod.Name, strategyStr)
		strategy := wfv1.ArtifactGCStrategy(strategyStr)
		// make sure we didn't already process this one
		alreadyRecouped := woc.wf.Status.ArtifactGCStatus.IsArtifactGCPodRecouped(pod.Name)
		if found && alreadyRecouped {
			// already processed
			fmt.Printf("deletethis: pod %s for strategy %s was already recouped\n", pod.Name, strategyStr)
			continue
		}

		phase := pod.Status.Phase
		woc.log.WithField("pod", pod.Name).
			WithField("phase", phase).
			WithField("message", pod.Status.Message).
			Info("reconciling artifact-gc pod")

		if phase == corev1.PodSucceeded || phase == corev1.PodFailed {
			err = woc.processCompletedArtifactGCPod(ctx, pod, strategy)
			if err != nil {
				return err
			}
			woc.wf.Status.ArtifactGCStatus.SetArtifactGCPodRecouped(pod.Name, true)
			if phase == corev1.PodSucceeded {
				anyPodSuccess = true
			}
			woc.updated = true
		}
	}

	if anyPodSuccess {
		// check if all artifacts have been deleted and if so remove Finalizer
		if woc.allArtifactsDeleted() {
			woc.log.Info("no remaining artifacts to GC, removing artifact GC finalizer")
			woc.wf.Finalizers = slice.RemoveString(woc.wf.Finalizers, common.FinalizerArtifactGC)
			woc.updated = true
		}

	}
	return nil
}

func (woc *wfOperationCtx) allArtifactsDeleted() bool {
	for _, n := range woc.wf.Status.Nodes {
		for _, a := range n.GetOutputs().GetArtifacts() {
			if !a.Deleted {
				return false
			}
		}
	}
	return true
}

func (woc *wfOperationCtx) processCompletedArtifactGCPod(ctx context.Context, pod *corev1.Pod, strategy wfv1.ArtifactGCStrategy) error {
	woc.log.Infof("processing completed Artifact GC Pod '%s'", pod.Name)

	// get associated WorkflowArtifactGCTasks
	labelSelector := fmt.Sprintf("%s = %s", common.LabelKeyArtifactGCPodName, pod.Name)
	taskList, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowArtifactGCTasks(woc.wf.Namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("failed to List WorkflowArtifactGCTasks: %w", err)
	}

	for _, task := range taskList.Items {
		// for each WorkflowArtifactGCTask: call processCompletedWorkflowArtifactGCTask() which can delete the Task and also should return whether there was an error
		err = woc.processCompletedWorkflowArtifactGCTask(ctx, &task, strategy)
		if err != nil {
			return err
		}
	}
	return nil
}

// process the Status in the WorkflowArtifactGCTask which was completed and reflect it in Workflow Status; then delete the Task CRD Object
// return first found error message if GC failed
func (woc *wfOperationCtx) processCompletedWorkflowArtifactGCTask(ctx context.Context, artifactGCTask *wfv1.WorkflowArtifactGCTask, strategy wfv1.ArtifactGCStrategy) error {
	woc.log.Debugf("processing WorkflowArtifactGCTask %s", artifactGCTask.Name)

	foundGCFailure := false
	for nodeName, nodeResult := range artifactGCTask.Status.ArtifactResultsByNode {
		// find this node result in the Workflow Status
		wfNode, found := woc.wf.Status.Nodes[nodeName]
		if !found {
			return fmt.Errorf("node named '%s' returned by WorkflowArtifactGCTask '%s' wasn't found in Workflow '%s' Status", nodeName, artifactGCTask.Name, woc.wf.Name)
		}

		if wfNode.Outputs == nil {
			return fmt.Errorf("node named '%s' returned by WorkflowArtifactGCTask '%s' doesn't seem to have Outputs in Workflow Status")
		}
		for i, wfArtifact := range wfNode.Outputs.Artifacts {
			// find artifact in the WorkflowArtifactGCTask Status
			artifactResult, foundArt := nodeResult.ArtifactResults[wfArtifact.Name]
			if !foundArt {
				// could be in a different WorkflowArtifactGCTask
				continue
			}
			fmt.Printf("deletethis: setting artifact Deleted=%t, %+v\n", artifactResult.Success, woc.wf.Status.Nodes[nodeName].Outputs.Artifacts[i])
			woc.wf.Status.Nodes[nodeName].Outputs.Artifacts[i].Deleted = artifactResult.Success
			if artifactResult.Error != nil {
				// issue an Event if there was an error - just do this one to prevent flooding the system with Events
				if !foundGCFailure {
					foundGCFailure = true
					gcFailureMsg := *artifactResult.Error
					woc.eventRecorder.Event(woc.wf, apiv1.EventTypeWarning, "ArtifactGCFailed",
						fmt.Sprintf("Artifact Garbage Collection failed for strategy %s, err:%s", strategy, gcFailureMsg))
				}
			}
		}

	}

	// now we can delete it
	// todo: temporarily commented out for testing
	//woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowArtifactGCTasks(woc.wf.Namespace).Delete(ctx, artifactGCTask.Name, metav1.DeleteOptions{})

	return nil
}

func (woc *wfOperationCtx) getArtifactAccess(artifact *wfv1.Artifact) podInfo {
	//  start with Workflow.ArtifactGC and override with Artifact.ArtifactGC
	podAccessInfo := podInfo{}
	if woc.wf.Spec.ArtifactGC != nil {
		woc.updateArtifactAccess(woc.wf.Spec.ArtifactGC, &podAccessInfo)
	}
	if artifact.ArtifactGC != nil {
		woc.updateArtifactAccess(artifact.ArtifactGC, &podAccessInfo)
	}
	return podAccessInfo
}

func (woc *wfOperationCtx) updateArtifactAccess(artifactGC *wfv1.ArtifactGC, podAccessInfo *podInfo) {

	if artifactGC.ServiceAccountName != "" {
		podAccessInfo.serviceAccount = artifactGC.ServiceAccountName
	}
	if artifactGC.PodMetadata != nil {
		if len(artifactGC.PodMetadata.Labels) > 0 && podAccessInfo.podMetadata.Labels == nil {
			podAccessInfo.podMetadata.Labels = make(map[string]string)
		}
		for labelKey, labelValue := range artifactGC.PodMetadata.Labels {
			podAccessInfo.podMetadata.Labels[labelKey] = labelValue
		}
		if len(artifactGC.PodMetadata.Annotations) > 0 && podAccessInfo.podMetadata.Annotations == nil {
			podAccessInfo.podMetadata.Annotations = make(map[string]string)
		}
		for annotationKey, annotationValue := range artifactGC.PodMetadata.Annotations {
			podAccessInfo.podMetadata.Annotations[annotationKey] = annotationValue
		}
	}

}

type templatesToArtifacts map[string]wfv1.ArtifactSearchResults

func (woc *wfOperationCtx) processArtifactGCStrategy(ctx context.Context, strategy wfv1.ArtifactGCStrategy) error {

	defer func() {
		woc.wf.Status.ArtifactGCStatus.SetArtifactGCStrategyProcessed(strategy, true)
		woc.updated = true
	}()

	// determine current Status associated with this garbage collection strategy: has it run before?
	// If so, we don't want to run it again
	started := woc.wf.Status.ArtifactGCStatus.IsArtifactGCStrategyProcessed(strategy)
	if started {
		return nil
	}

	var err error

	woc.log.Debugf("processing Artifact GC Strategy %s", strategy)

	// Search for artifacts
	artifactSearchResults := woc.execWf.SearchArtifacts(&wfv1.ArtifactSearchQuery{ArtifactGCStrategies: map[wfv1.ArtifactGCStrategy]bool{strategy: true}, Deleted: pointer.BoolPtr(false)})
	if len(artifactSearchResults) == 0 {
		woc.log.Debugf("No Artifact Search Results returned from strategy %s", strategy)
		return nil
	}

	// cache the templates by name so we can find them easily
	templatesByName := make(map[string]*wfv1.Template)

	// We need to create a separate Pod for each set of Artifacts that require special access requirements (i.e. Service Account and Pod Metadata)
	// So first group artifacts that need to be deleted by access requirements

	// grouping the Artifacts according to access requirements and Template
	groupedByPod := make(map[string]templatesToArtifacts)

	// a mapping from the name we'll use for the Pod to the actual metadata and Service Account that need to be applied for that Pod
	podNames := make(map[string]podInfo)

	var podName string
	var podAccessInfo podInfo

	for _, artifactSearchResult := range artifactSearchResults {
		// get the access requirements and hash them
		podAccessInfo = woc.getArtifactAccess(&artifactSearchResult.Artifact)
		podName = woc.artGCPodName(strategy, podAccessInfo)
		_, found := podNames[podName]
		if !found {
			podNames[podName] = podAccessInfo
		}
		_, found = groupedByPod[podName]
		if !found {
			groupedByPod[podName] = make(templatesToArtifacts)
		}
		// get the Template
		node, found := woc.wf.Status.Nodes[artifactSearchResult.NodeID]
		if !found {
			//todo: error
		}
		template, found := templatesByName[node.TemplateName]
		if !found {
			template = woc.wf.GetTemplateByName(node.TemplateName)
			if template == nil {
				//todo: error
			}
			templatesByName[node.TemplateName] = template
		}

		_, found = groupedByPod[podName][template.Name]
		if !found {
			groupedByPod[podName][template.Name] = make(wfv1.ArtifactSearchResults, 0)
		}

		groupedByPod[podName][template.Name] = append(groupedByPod[podName][template.Name], artifactSearchResult)
	}

	fmt.Printf("deletethis: groupedByPod=%+v\n", groupedByPod)

	// start up a separate Pod with a separate set of ArtifactGCTasks for it to use for each unique Access Requirement
	for podName, templatesToArtList := range groupedByPod {
		tasks := make([]*wfv1.WorkflowArtifactGCTask, 0)

		fmt.Printf("deletethis: processing podName %s from groupedByPod\n", podName)

		for templateName, artifacts := range templatesToArtList {

			fmt.Printf("deletethis: for podName '%s' from groupedByPod, processing templateName '%s'\n", podName, templateName)
			template := templatesByName[templateName]
			woc.addTemplateArtifactsToTasks(strategy, podName, &tasks, template, artifacts)
		}

		if len(tasks) > 0 {
			// create the K8s WorkflowArtifactGCTask objects
			for i, task := range tasks {
				tasks[i], err = woc.createWorkflowArtifactGCTask(ctx, task)
				if err != nil {
					return err
				}
			}
			// create the pod
			podAccessInfo, found := podNames[podName]
			if !found {
				//todo: error
			}
			_, err := woc.createArtifactGCPod(ctx, strategy, tasks, podAccessInfo, podName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type podInfo struct {
	serviceAccount string
	podMetadata    wfv1.Metadata
}

/*
// create a string to uniquely identify a pod by its access requirement
func (woc *wfOperationCtx) getPodIDHash(podAccessInfo podInfo) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(podAccessInfo.serviceAccount))
	// we should be able to always get the same result regardless of the order of our Labels or Annotations
	// so sort alphabetically
	sortedLabels := maps.Keys(podAccessInfo.podMetadata.Labels)
	sort.Strings(sortedLabels)
	for _, label := range sortedLabels {
		labelValue := podAccessInfo.podMetadata.Labels[label]
		_, _ = h.Write([]byte(label))
		_, _ = h.Write([]byte(labelValue))
	}

	sortedAnnotations := maps.Keys(podAccessInfo.podMetadata.Annotations)
	sort.Strings(sortedAnnotations)
	for _, annotation := range sortedAnnotations {
		annotationValue := podAccessInfo.podMetadata.Annotations[annotation]
		_, _ = h.Write([]byte(annotation))
		_, _ = h.Write([]byte(annotationValue))
	}
	return fmt.Sprintf("%v", h.Sum32())
}


func (woc *wfOperationCtx) artGCPodName(strategy wfv1.ArtifactGCStrategy, podIDHash string) string {
	return fmt.Sprintf("%s-artgc-%s-%s", woc.wf.Name, strategy.AbbreviatedName(), podIDHash)
}*/

func (woc *wfOperationCtx) artGCPodName(strategy wfv1.ArtifactGCStrategy, podAccessInfo podInfo) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(podAccessInfo.serviceAccount))
	// we should be able to always get the same result regardless of the order of our Labels or Annotations
	// so sort alphabetically
	sortedLabels := maps.Keys(podAccessInfo.podMetadata.Labels)
	sort.Strings(sortedLabels)
	for _, label := range sortedLabels {
		labelValue := podAccessInfo.podMetadata.Labels[label]
		_, _ = h.Write([]byte(label))
		_, _ = h.Write([]byte(labelValue))
	}

	sortedAnnotations := maps.Keys(podAccessInfo.podMetadata.Annotations)
	sort.Strings(sortedAnnotations)
	for _, annotation := range sortedAnnotations {
		annotationValue := podAccessInfo.podMetadata.Annotations[annotation]
		_, _ = h.Write([]byte(annotation))
		_, _ = h.Write([]byte(annotationValue))
	}

	return fmt.Sprintf("%s-artgc-%s-%v", woc.wf.Name, strategy.AbbreviatedName(), h.Sum32())
}

func (woc *wfOperationCtx) artGCTaskName(podName string, taskIndex int) string {
	return fmt.Sprintf("%s-%d", podName, taskIndex)
}

func (woc *wfOperationCtx) addTemplateArtifactsToTasks(strategy wfv1.ArtifactGCStrategy, podName string, tasks *[]*wfv1.WorkflowArtifactGCTask, template *wfv1.Template, artifactSearchResults wfv1.ArtifactSearchResults) {
	if len(artifactSearchResults) == 0 {
		return
	}
	if tasks == nil {
		ts := make([]*wfv1.WorkflowArtifactGCTask, 0)
		tasks = &ts
	}

	// do we need to generate a new WorkflowArtifactGCTask or can we use current?
	//if len(tasks) == 0 || tasks[len(tasks) - 1].Spec.Tasks
	if len(*tasks) == 0 {
		currentTask := &wfv1.WorkflowArtifactGCTask{
			TypeMeta: metav1.TypeMeta{
				Kind:       workflow.WorkflowArtifactGCTaskKind,
				APIVersion: workflow.APIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: woc.wf.Namespace,
				Name:      woc.artGCTaskName(podName, 0),
				Labels:    map[string]string{common.LabelKeyArtifactGCPodName: podName},
				OwnerReferences: []metav1.OwnerReference{ // make sure we get deleted with the workflow
					*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
				},
			},
			Spec: wfv1.ArtifactGCSpec{
				ArtifactsByNode: make(map[string]wfv1.ArtifactNodeSpec),
			},
		}
		*tasks = append(*tasks, currentTask)
	} /*else if len(*tasks) > SOME_THRESHOLD { //todo: handle multiple WorkflowArtifactGCTasks
		// add a new WorkflowArtifactGCTask to *tasks
	}*/

	currentTask := (*tasks)[len(*tasks)-1]
	artifactsByNode := currentTask.Spec.ArtifactsByNode

	archiveLocation := template.ArchiveLocation
	if archiveLocation == nil {
		archiveLocation = woc.artifactRepository.ToArtifactLocation()
	}

	// go through artifactSearchResults and create a map from nodeID to artifacts
	// for each node, create an ArtifactNodeSpec with our Template's ArchiveLocation (if any) and our list of Artifacts
	for _, artifactSearchResult := range artifactSearchResults {
		artifactNodeSpec, found := artifactsByNode[artifactSearchResult.NodeID]
		if !found {
			artifactsByNode[artifactSearchResult.NodeID] = wfv1.ArtifactNodeSpec{
				ArchiveLocation: archiveLocation,
				Artifacts:       make(map[string]wfv1.Artifact),
			}
			artifactNodeSpec = artifactsByNode[artifactSearchResult.NodeID]
		}

		artifactNodeSpec.Artifacts[artifactSearchResult.Name] = artifactSearchResult.Artifact

	}
	woc.log.Debugf("list of artifacts pertaining to template %s to WorkflowArtifactGCTask '%s': %+v", template.Name, currentTask.Name, artifactsByNode)

}

func (woc *wfOperationCtx) getArtifactTask(taskName string) (*wfv1.WorkflowArtifactGCTask, error) {
	key := woc.wf.Namespace + "/" + taskName
	task, exists, err := woc.controller.artGCTaskInformer.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get WorkflowArtifactGCTask by key '%s': %w", key, err)
	}
	if !exists {
		return nil, nil
	}
	return task.(*wfv1.WorkflowArtifactGCTask), nil
}

//	create WorkflowArtifactGCTask CRD object
func (woc *wfOperationCtx) createWorkflowArtifactGCTask(ctx context.Context, task *wfv1.WorkflowArtifactGCTask) (*wfv1.WorkflowArtifactGCTask, error) {

	// first make sure it doesn't already exist
	foundTask, err := woc.getArtifactTask(task.Name)
	if err != nil {
		return nil, err
	}
	if foundTask != nil {
		woc.log.Debugf("Artifact GC Task %s already exists", task.Name)
	} else {
		woc.log.Infof("Creating Artifact GC Task %s", task.Name)

		task, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowArtifactGCTasks(woc.wf.Namespace).Create(ctx, task, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to Create WorkflowArtifactGCTask '%s' for Garbage Collection: %w", task.Name, err)
		}
	}
	return task, nil
}

func (woc *wfOperationCtx) createArtifactGCPod(ctx context.Context, strategy wfv1.ArtifactGCStrategy, tasks []*wfv1.WorkflowArtifactGCTask, podAccessInfo podInfo, podName string) (*corev1.Pod, error) {

	woc.log.
		WithField("strategy", strategy).
		Infof("create pod to delete artifacts: %s", podName)

	ownerReferences := make([]metav1.OwnerReference, len(tasks))
	for i, task := range tasks {
		// make sure pod gets deleted with the WorkflowArtifactGCTasks
		ownerReferences[i] = *metav1.NewControllerRef(task, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowArtifactGCTaskKind))
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name,
				common.LabelKeyComponent: artifactGCComponent,
				common.LabelKeyCompleted: "false", // todo: do we need this? what is the effect?
			},
			Annotations: map[string]string{
				common.AnnotationKeyArtifactGCStrategy: string(strategy),
			},

			OwnerReferences: ownerReferences,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            common.MainContainerName,
					Image:           woc.controller.executorImage(),
					ImagePullPolicy: woc.controller.executorImagePullPolicy(),
					Args:            []string{"artifact", "delete", "--loglevel", getExecutorLogLevel()},
					Env: []corev1.EnvVar{
						{Name: common.EnvVarArtifactGCPod, Value: podName},
					},
					// if this pod is breached by an attacker we:
					// * prevent installation of any new packages
					// * modification of the file-system
					SecurityContext: &corev1.SecurityContext{
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						Privileged:               pointer.Bool(false),
						RunAsNonRoot:             pointer.Bool(true),
						RunAsUser:                pointer.Int64Ptr(8737), //todo: magic number
						ReadOnlyRootFilesystem:   pointer.Bool(true),
						AllowPrivilegeEscalation: pointer.Bool(false),
					},
					// if this pod is breached by an attacker these limits prevent excessive CPU and memory usage
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("100m"), //todo: should these values be in the Controller config, and also maybe increased?
							"memory": resource.MustParse("64Mi"),
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("50m"),
							"memory": resource.MustParse("32Mi"),
						},
					},
				},
			},
			AutomountServiceAccountToken: pointer.Bool(true),
			RestartPolicy:                corev1.RestartPolicyOnFailure, //todo: verify
		},
	}

	// Use the Service Account and/or Labels and Annotations specified for Artifact GC, if they exist
	if podAccessInfo.serviceAccount != "" {
		pod.Spec.ServiceAccountName = podAccessInfo.serviceAccount
	}
	for label, labelVal := range podAccessInfo.podMetadata.Labels {
		pod.ObjectMeta.Labels[label] = labelVal
	}
	for annotation, annotationVal := range podAccessInfo.podMetadata.Annotations {
		pod.ObjectMeta.Labels[annotation] = annotationVal
	}

	if v := woc.controller.Config.InstanceID; v != "" {
		pod.Labels[common.EnvVarInstanceID] = v
	}

	_, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	fmt.Printf("deletethis: attempted to create GC pod of name %s: err=%v\n ", pod.Name, err)

	if err != nil {
		if apierr.IsAlreadyExists(err) {
			woc.log.Warningf("Artifact GC Pod %s already exists?", pod.Name)
		} else {
			return nil, fmt.Errorf("failed to create pod: %w", err)
		}
	}
	return pod, nil
}

/*
func (woc *wfOperationCtx) setArtifactGCPodAccess(pod *corev1.Pod, artifactGCPodSpec *wfv1.ArtifactGCPod) {
	if artifactGCPodSpec.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = artifactGCPodSpec.ServiceAccountName
	}
	if artifactGCPodSpec.PodMetadata != nil {
		for label, value := range artifactGCPodSpec.PodMetadata.Labels {
			pod.ObjectMeta.Labels[label] = value
		}
		for annotation, value := range artifactGCPodSpec.PodMetadata.Annotations {
			pod.ObjectMeta.Annotations[annotation] = value
		}
	}
}*/

/*

func (woc *wfOperationCtx) createArtifactGCPod(ctx context.Context, strategy wfv1.ArtifactGCStrategy, serviceAcct string, artifacts wfv1.ArtifactSearchResults, templates []*wfv1.Template) error {

	//	create a WorkflowArtifactGCTask which contains a subset of our Workflow's spec (just the Templates for this Service Account), as well as a GC Pod with the following environment variables:
	//		- WorkflowArtifactGCTask name
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

	taskName := podName
	err = woc.createWorkflowArtifactGCTaskSet(ctx, templates, taskName)
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
/*
	woc.log.
		WithField("strategy", strategy).
		WithField("serviceAcct", serviceAcct).
		Infof("create pod to delete artifacts: %s", podName)
*/
/*data, err := json.Marshal(a)
if err != nil {
	return fmt.Errorf("failed to marshall artifact: %w", err)
}*/
/*
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
/*
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
						{Name: common.EnvVarWorkflowArtifactGCTaskSet, Value: taskName},
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
*/
