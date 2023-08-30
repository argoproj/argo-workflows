package controller

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"

	"golang.org/x/exp/maps"
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
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

const artifactGCComponent = "artifact-gc"

// artifactGCEnabled is a feature flag to globally disabled artifact GC in case of emergency
var artifactGCEnabled, _ = env.GetBool("ARGO_ARTIFACT_GC_ENABLED", true)

func (woc *wfOperationCtx) garbageCollectArtifacts(ctx context.Context) error {

	if !artifactGCEnabled {
		return nil
	}

	if woc.wf.Status.ArtifactGCStatus == nil {
		woc.wf.Status.ArtifactGCStatus = &wfv1.ArtGCStatus{}
	}

	// only do Artifact GC if we have a Finalizer for it (i.e. Artifact GC is configured for this Workflow
	// and there's work left to do for it)
	if !slice.ContainsString(woc.wf.Finalizers, common.FinalizerArtifactGC) {
		if woc.wf.Status.ArtifactGCStatus.NotSpecified {
			return nil // we already verified it's not required for this workflow
		}
		if woc.HasArtifactGC() {
			woc.log.Info("adding artifact GC finalizer")
			finalizers := append(woc.wf.GetFinalizers(), common.FinalizerArtifactGC)
			woc.wf.SetFinalizers(finalizers)
			woc.wf.Status.ArtifactGCStatus.NotSpecified = false
		} else {
			woc.wf.Status.ArtifactGCStatus.NotSpecified = true
		}
		return nil
	}

	// based on current state of Workflow, which Artifact GC Strategies can be processed now?
	strategies := woc.artifactGCStrategiesReady()
	for strategy := range strategies {
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

func (woc *wfOperationCtx) HasArtifactGC() bool {
	// ArtifactGC can be defined on the Workflow level or on the Artifact level
	// It may be defined in the Workflow itself or in a WorkflowTemplate referenced by the Workflow

	// woc.execWf.Spec.Templates includes templates described directly in the Workflow as well as templates
	// in a WorkflowTemplate that the entire Workflow is based on
	for _, template := range woc.execWf.Spec.Templates {
		for _, artifact := range template.Outputs.Artifacts {
			strategy := woc.execWf.GetArtifactGCStrategy(&artifact)
			if strategy != wfv1.ArtifactGCStrategyUndefined && strategy != wfv1.ArtifactGCNever {
				return true
			}
		}
	}

	// need to go to woc.wf.Status.StoredTemplates in the case of a Step referencing a WorkflowTemplate
	for _, template := range woc.wf.Status.StoredTemplates {
		for _, artifact := range template.Outputs.Artifacts {
			strategy := woc.execWf.GetArtifactGCStrategy(&artifact)
			if strategy != wfv1.ArtifactGCStrategyUndefined && strategy != wfv1.ArtifactGCNever {
				return true
			}
		}
	}

	return false
}

// which ArtifactGC Strategies are ready to process?
func (woc *wfOperationCtx) artifactGCStrategiesReady() map[wfv1.ArtifactGCStrategy]struct{} {
	strategies := map[wfv1.ArtifactGCStrategy]struct{}{} // essentially a Set

	if woc.wf.Labels[common.LabelKeyCompleted] == "true" || woc.wf.DeletionTimestamp != nil {
		if !woc.wf.Status.ArtifactGCStatus.IsArtifactGCStrategyProcessed(wfv1.ArtifactGCOnWorkflowCompletion) {
			strategies[wfv1.ArtifactGCOnWorkflowCompletion] = struct{}{}
		}
	}
	if woc.wf.DeletionTimestamp != nil {
		if !woc.wf.Status.ArtifactGCStatus.IsArtifactGCStrategyProcessed(wfv1.ArtifactGCOnWorkflowDeletion) {
			strategies[wfv1.ArtifactGCOnWorkflowDeletion] = struct{}{}
		}
	}

	return strategies
}

type templatesToArtifacts map[string]wfv1.ArtifactSearchResults

// Artifact GC Strategy is ready: start up Pods to handle it
func (woc *wfOperationCtx) processArtifactGCStrategy(ctx context.Context, strategy wfv1.ArtifactGCStrategy) error {

	defer func() {
		woc.wf.Status.ArtifactGCStatus.SetArtifactGCStrategyProcessed(strategy, true)
		woc.updated = true
	}()

	var err error

	woc.log.Debugf("processing Artifact GC Strategy %s", strategy)

	// Search for artifacts
	artifactSearchResults := woc.findArtifactsToGC(strategy)
	if len(artifactSearchResults) == 0 {
		woc.log.Debugf("No Artifact Search Results returned from strategy %s", strategy)
		return nil
	}

	// cache the templates by name so we can find them easily
	templatesByName := make(map[string]*wfv1.Template)

	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// We need to create a separate Pod for each set of Artifacts that require special permissions
	// (i.e. Service Account and Pod Metadata)
	// So first group artifacts that need to be deleted by permissions
	/////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	groupedByPod := make(map[string]templatesToArtifacts)

	// a mapping from the name we'll use for the Pod to the actual metadata and Service Account that need to be applied for that Pod
	podNames := make(map[string]podInfo)

	var podName string
	var podInfo podInfo

	for _, artifactSearchResult := range artifactSearchResults {
		// get the permissions required for this artifact and create a unique Pod name from them
		podInfo = woc.getArtifactGCPodInfo(&artifactSearchResult.Artifact)
		podName, err = woc.artGCPodName(strategy, podInfo)
		if err != nil {
			return err
		}
		if _, found := podNames[podName]; !found {
			podNames[podName] = podInfo
		}
		if _, found := groupedByPod[podName]; !found {
			groupedByPod[podName] = make(templatesToArtifacts)
		}
		// get the Template for the Artifact
		node, err := woc.wf.Status.Nodes.Get(artifactSearchResult.NodeID)
		if err != nil {
			woc.log.Errorf("Was unable to obtain node for %s", artifactSearchResult.NodeID)
			return fmt.Errorf("can't process Artifact GC Strategy %s: node ID %q not found in Status??", strategy, artifactSearchResult.NodeID)
		}
		templateName := node.TemplateName
		if templateName == "" && node.GetTemplateRef() != nil {
			templateName = node.GetTemplateRef().Template
		}
		if templateName == "" {
			return fmt.Errorf("can't process Artifact GC Strategy %s: node %+v has an unnamed template", strategy, node)
		}
		template, found := templatesByName[templateName]
		if !found {
			template = woc.wf.GetTemplateByName(templateName)
			if template == nil {
				return fmt.Errorf("can't process Artifact GC Strategy %s: template name %q belonging to node %+v not found??", strategy, templateName, node)
			}
			templatesByName[templateName] = template
		}

		if _, found := groupedByPod[podName][template.Name]; !found {
			groupedByPod[podName][template.Name] = make(wfv1.ArtifactSearchResults, 0)
		}

		groupedByPod[podName][template.Name] = append(groupedByPod[podName][template.Name], artifactSearchResult)
	}

	// start up a separate Pod with a separate set of ArtifactGCTasks for it to use for each unique Service Account/metadata
	for podName, templatesToArtList := range groupedByPod {
		tasks := make([]*wfv1.WorkflowArtifactGCTask, 0)

		for templateName, artifacts := range templatesToArtList {
			template := templatesByName[templateName]
			woc.addTemplateArtifactsToTasks(podName, &tasks, template, artifacts)
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
				return fmt.Errorf("can't find podInfo for podName %q??", podName)
			}

			_, err := woc.createArtifactGCPod(ctx, strategy, tasks, podAccessInfo, podName, templatesToArtList, templatesByName)
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
	podSpecPatch   string
}

// get Pod name
// (we have a unique Pod for each Artifact GC Strategy and Service Account/Metadata requirement)
func (woc *wfOperationCtx) artGCPodName(strategy wfv1.ArtifactGCStrategy, podInfo podInfo) (string, error) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(podInfo.serviceAccount))
	// we should be able to always get the same result regardless of the order of our Labels or Annotations
	// so sort alphabetically
	sortedLabels := maps.Keys(podInfo.podMetadata.Labels)
	sort.Strings(sortedLabels)
	for _, label := range sortedLabels {
		labelValue := podInfo.podMetadata.Labels[label]
		_, _ = h.Write([]byte(label))
		_, _ = h.Write([]byte(labelValue))
	}

	sortedAnnotations := maps.Keys(podInfo.podMetadata.Annotations)
	sort.Strings(sortedAnnotations)
	for _, annotation := range sortedAnnotations {
		annotationValue := podInfo.podMetadata.Annotations[annotation]
		_, _ = h.Write([]byte(annotation))
		_, _ = h.Write([]byte(annotationValue))
	}

	abbreviatedName := ""
	switch strategy {
	case wfv1.ArtifactGCOnWorkflowCompletion:
		abbreviatedName = "wfcomp"
	case wfv1.ArtifactGCOnWorkflowDeletion:
		abbreviatedName = "wfdel"
	default:
		return "", fmt.Errorf("ArtifactGCStrategy %q not valid", strategy)
	}

	return fmt.Sprintf("%s-artgc-%s-%v", woc.wf.Name, abbreviatedName, h.Sum32()), nil
}

func (woc *wfOperationCtx) artGCTaskName(podName string, taskIndex int) string {
	return fmt.Sprintf("%s-%d", podName, taskIndex)
}

func (woc *wfOperationCtx) artifactGCPodLabel(podName string) string {
	hashedPod := fnv.New32a()
	_, _ = hashedPod.Write([]byte(podName))
	return fmt.Sprintf("%d", hashedPod.Sum32())
}

func (woc *wfOperationCtx) addTemplateArtifactsToTasks(podName string, tasks *[]*wfv1.WorkflowArtifactGCTask, template *wfv1.Template, artifactSearchResults wfv1.ArtifactSearchResults) {
	if len(artifactSearchResults) == 0 {
		return
	}

	if tasks == nil {
		ts := make([]*wfv1.WorkflowArtifactGCTask, 0)
		tasks = &ts
	}

	// do we need to generate a new WorkflowArtifactGCTask or can we use current?
	// todo: currently we're only handling one but may require more in the future if we start to reach 1 MB in the CRD
	if len(*tasks) == 0 {
		currentTask := &wfv1.WorkflowArtifactGCTask{
			TypeMeta: metav1.TypeMeta{
				Kind:       workflow.WorkflowArtifactGCTaskKind,
				APIVersion: workflow.APIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: woc.wf.Namespace,
				Name:      woc.artGCTaskName(podName, 0),
				Labels:    map[string]string{common.LabelKeyArtifactGCPodHash: woc.artifactGCPodLabel(podName)},
				OwnerReferences: []metav1.OwnerReference{ // make sure we get deleted with the workflow
					*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
				},
			},
			Spec: wfv1.ArtifactGCSpec{
				ArtifactsByNode: make(map[string]wfv1.ArtifactNodeSpec),
			},
		}
		*tasks = append(*tasks, currentTask)
	} /*else if hitting 1 MB on CRD { //todo: handle multiple WorkflowArtifactGCTasks
		// add a new WorkflowArtifactGCTask to *tasks
	}*/

	currentTask := (*tasks)[len(*tasks)-1]
	artifactsByNode := currentTask.Spec.ArtifactsByNode

	// if ArchiveLocation is specified for the Template use that, otherwise use default
	archiveLocation := template.ArchiveLocation
	if !archiveLocation.HasLocation() {
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
	woc.log.Debugf("list of artifacts pertaining to template %s to WorkflowArtifactGCTask %q: %+v", template.Name, currentTask.Name, artifactsByNode)

}

// find WorkflowArtifactGCTask CRD object by name
func (woc *wfOperationCtx) getArtifactTask(taskName string) (*wfv1.WorkflowArtifactGCTask, error) {
	key := woc.wf.Namespace + "/" + taskName
	task, exists, err := woc.controller.artGCTaskInformer.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get WorkflowArtifactGCTask by key %q: %w", key, err)
	}
	if !exists {
		return nil, nil
	}
	return task.(*wfv1.WorkflowArtifactGCTask), nil
}

// create WorkflowArtifactGCTask CRD object
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
			return nil, fmt.Errorf("failed to Create WorkflowArtifactGCTask %q for Garbage Collection: %w", task.Name, err)
		}
	}
	return task, nil
}

// create the Pod which will do the deletions
func (woc *wfOperationCtx) createArtifactGCPod(ctx context.Context, strategy wfv1.ArtifactGCStrategy, tasks []*wfv1.WorkflowArtifactGCTask,
	podInfo podInfo, podName string, templatesToArtList templatesToArtifacts, templatesByName map[string]*wfv1.Template) (*corev1.Pod, error) {

	woc.log.
		WithField("strategy", strategy).
		Infof("creating pod to delete artifacts: %s", podName)

	// Pod is owned by WorkflowArtifactGCTasks, so it will die automatically when all of them have died
	ownerReferences := make([]metav1.OwnerReference, len(tasks))
	for i, task := range tasks {
		// make sure pod gets deleted with the WorkflowArtifactGCTasks
		ownerReferences[i] = *metav1.NewControllerRef(task, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowArtifactGCTaskKind))
	}

	artifactLocations := make([]*wfv1.ArtifactLocation, 0)

	for templateName, artifacts := range templatesToArtList {
		template, found := templatesByName[templateName]
		if !found {
			return nil, fmt.Errorf("can't find template with name %s???", templateName)
		}

		if template.ArchiveLocation.HasLocation() {
			artifactLocations = append(artifactLocations, template.ArchiveLocation)
		} else {
			artifactLocations = append(artifactLocations, woc.artifactRepository.ToArtifactLocation())
		}
		for i := range artifacts {
			artifactLocations = append(artifactLocations, &artifacts[i].ArtifactLocation)
		}
	}

	volumes, volumeMounts := createSecretVolumesAndMountsFromArtifactLocations(artifactLocations)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name,
				common.LabelKeyComponent: artifactGCComponent,
				common.LabelKeyCompleted: "false",
			},
			Annotations: map[string]string{
				common.AnnotationKeyArtifactGCStrategy: string(strategy),
			},

			OwnerReferences: ownerReferences,
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
						{Name: common.EnvVarArtifactGCPodHash, Value: woc.artifactGCPodLabel(podName)},
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
			AutomountServiceAccountToken: pointer.Bool(true),
			RestartPolicy:                corev1.RestartPolicyNever,
		},
	}

	if podInfo.podSpecPatch != "" {
		patchedPodSpec, err := util.ApplyPodSpecPatch(pod.Spec, podInfo.podSpecPatch)
		if err != nil {
			return nil, err
		}
		pod.Spec = *patchedPodSpec
	}

	// Use the Service Account and/or Labels and Annotations specified for our Pod, if they exist
	if podInfo.serviceAccount != "" {
		pod.Spec.ServiceAccountName = podInfo.serviceAccount
	}
	for label, labelVal := range podInfo.podMetadata.Labels {
		pod.ObjectMeta.Labels[label] = labelVal
	}
	for annotation, annotationVal := range podInfo.podMetadata.Annotations {
		pod.ObjectMeta.Annotations[annotation] = annotationVal
	}

	if v := woc.controller.Config.InstanceID; v != "" {
		pod.Labels[common.EnvVarInstanceID] = v
	}

	_, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Create(ctx, pod, metav1.CreateOptions{})

	if err != nil {
		if apierr.IsAlreadyExists(err) {
			woc.log.Warningf("Artifact GC Pod %s already exists?", pod.Name)
		} else {
			return nil, fmt.Errorf("failed to create pod: %w", err)
		}
	}
	return pod, nil
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
		if pod.Labels[common.LabelKeyComponent] != artifactGCComponent { // make sure it's an Artifact GC Pod
			continue
		}

		// make sure we didn't already process this one
		if woc.wf.Status.ArtifactGCStatus.IsArtifactGCPodRecouped(pod.Name) {
			// already processed
			continue
		}

		phase := pod.Status.Phase

		// if Pod is done processing the results
		if phase == corev1.PodSucceeded || phase == corev1.PodFailed {
			woc.log.WithField("pod", pod.Name).
				WithField("phase", phase).
				WithField("message", pod.Status.Message).
				Info("reconciling artifact-gc pod")

			err = woc.processCompletedArtifactGCPod(ctx, pod)
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

	removeFinalizer := false
	forceFinalizerRemoval := woc.execWf.Spec.ArtifactGC != nil && woc.execWf.Spec.ArtifactGC.ForceFinalizerRemoval
	if forceFinalizerRemoval {
		removeFinalizer = woc.wf.Status.ArtifactGCStatus.AllArtifactGCPodsRecouped()
	} else {
		// check if all artifacts have been deleted and if so remove Finalizer
		removeFinalizer = anyPodSuccess && woc.allArtifactsDeleted()
	}
	if removeFinalizer {
		woc.log.Infof("no remaining artifacts to GC, removing artifact GC finalizer (forceFinalizerRemoval=%v)", forceFinalizerRemoval)
		woc.wf.Finalizers = slice.RemoveString(woc.wf.Finalizers, common.FinalizerArtifactGC)
		woc.updated = true
	}
	return nil
}

func (woc *wfOperationCtx) allArtifactsDeleted() bool {
	for _, n := range woc.wf.Status.Nodes {
		if n.Type != wfv1.NodeTypePod {
			continue
		}
		for _, a := range n.GetOutputs().GetArtifacts() {
			if !a.Deleted && woc.execWf.GetArtifactGCStrategy(&a) != wfv1.ArtifactGCNever && woc.execWf.GetArtifactGCStrategy(&a) != wfv1.ArtifactGCStrategyUndefined {
				return false
			}
		}
	}
	return true
}

func (woc *wfOperationCtx) findArtifactsToGC(strategy wfv1.ArtifactGCStrategy) wfv1.ArtifactSearchResults {

	var results wfv1.ArtifactSearchResults

	for _, n := range woc.wf.Status.Nodes {

		if n.Type != wfv1.NodeTypePod {
			continue
		}
		for _, a := range n.GetOutputs().GetArtifacts() {

			// artifact strategy is either based on overall Workflow ArtifactGC Strategy, or
			// if it's specified on the individual artifact level that takes priority
			artifactStrategy := woc.execWf.GetArtifactGCStrategy(&a)
			if artifactStrategy == strategy && !a.Deleted {
				results = append(results, wfv1.ArtifactSearchResult{Artifact: a, NodeID: n.ID})
			}
		}
	}
	return results
}

func (woc *wfOperationCtx) processCompletedArtifactGCPod(ctx context.Context, pod *corev1.Pod) error {
	woc.log.Infof("processing completed Artifact GC Pod %q", pod.Name)

	strategyStr, found := pod.Annotations[common.AnnotationKeyArtifactGCStrategy]
	if !found {
		return fmt.Errorf("Artifact GC Pod %q doesn't have annotation %q?", pod.Name, common.AnnotationKeyArtifactGCStrategy)
	}
	strategy := wfv1.ArtifactGCStrategy(strategyStr)

	if pod.Status.Phase == corev1.PodFailed {
		errMsg := fmt.Sprintf("Artifact Garbage Collection failed for strategy %s, pod %s exited with non-zero exit code: check pod logs for more information", pod.Name, strategy)
		woc.addArtGCCondition(errMsg)
		woc.addArtGCEvent(errMsg)
	}

	// get associated WorkflowArtifactGCTasks
	labelSelector := fmt.Sprintf("%s = %s", common.LabelKeyArtifactGCPodHash, woc.artifactGCPodLabel(pod.Name))
	taskList, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowArtifactGCTasks(woc.wf.Namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("failed to List WorkflowArtifactGCTasks: %w", err)
	}

	for _, task := range taskList.Items {
		allArtifactsSucceeded, err := woc.processCompletedWorkflowArtifactGCTask(&task, strategy)
		if err != nil {
			return err
		}
		if allArtifactsSucceeded && pod.Status.Phase == corev1.PodSucceeded {
			// now we can delete it, if it succeeded (otherwise we leave it up to be inspected)
			woc.log.Debugf("deleting WorkflowArtifactGCTask: %s", task.Name)
			err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowArtifactGCTasks(woc.wf.Namespace).Delete(ctx, task.Name, metav1.DeleteOptions{})
			if err != nil {
				woc.log.Errorf("error deleting WorkflowArtifactGCTask: %s: %v", task.Name, err)
			}
		}

	}
	return nil
}

// process the Status in the WorkflowArtifactGCTask which was completed and reflect it in Workflow Status; then delete the Task CRD Object
// return true if all artifacts succeeded, else false
func (woc *wfOperationCtx) processCompletedWorkflowArtifactGCTask(artifactGCTask *wfv1.WorkflowArtifactGCTask, strategy wfv1.ArtifactGCStrategy) (bool, error) {
	woc.log.Debugf("processing WorkflowArtifactGCTask %s", artifactGCTask.Name)

	foundGCFailure := false
	for nodeName, nodeResult := range artifactGCTask.Status.ArtifactResultsByNode {
		// find this node result in the Workflow Status
		wfNode, err := woc.wf.Status.Nodes.Get(nodeName)
		if err != nil {
			woc.log.Errorf("Was unable to obtain node for %s", nodeName)
			return false, fmt.Errorf("node named %q returned by WorkflowArtifactGCTask %q wasn't found in Workflow %q Status", nodeName, artifactGCTask.Name, woc.wf.Name)
		}
		if wfNode.Outputs == nil {
			return false, fmt.Errorf("node named %q returned by WorkflowArtifactGCTask %q doesn't seem to have Outputs in Workflow Status", nodeName, artifactGCTask.Name)
		}
		for i, wfArtifact := range wfNode.Outputs.Artifacts {
			// find artifact in the WorkflowArtifactGCTask Status
			artifactResult, foundArt := nodeResult.ArtifactResults[wfArtifact.Name]
			if !foundArt {
				// could be in a different WorkflowArtifactGCTask
				continue
			}

			wfNode.Outputs.Artifacts[i].Deleted = artifactResult.Success
			woc.wf.Status.Nodes.Set(nodeName, *wfNode)

			if artifactResult.Error != nil {
				woc.addArtGCCondition(fmt.Sprintf("%s (artifactGCTask: %s)", *artifactResult.Error, artifactGCTask.Name))
				// issue an Event if there was an error - just do this once to prevent flooding the system with Events
				if !foundGCFailure {
					foundGCFailure = true
					gcFailureMsg := *artifactResult.Error
					woc.addArtGCEvent(fmt.Sprintf("Artifact Garbage Collection failed for strategy %s, err:%s", strategy, gcFailureMsg))
				}
			}
		}

	}

	return !foundGCFailure, nil
}

func (woc *wfOperationCtx) addArtGCCondition(msg string) {
	woc.wf.Status.Conditions.UpsertCondition(wfv1.Condition{
		Type:    wfv1.ConditionTypeArtifactGCError,
		Status:  metav1.ConditionTrue,
		Message: msg,
	})
}

func (woc *wfOperationCtx) addArtGCEvent(msg string) {
	woc.eventRecorder.Event(woc.wf, corev1.EventTypeWarning, "ArtifactGCFailed", msg)
}

func (woc *wfOperationCtx) getArtifactGCPodInfo(artifact *wfv1.Artifact) podInfo {
	//  start with Workflow.ArtifactGC and override with Artifact.ArtifactGC
	podInfo := podInfo{}
	if woc.execWf.Spec.ArtifactGC != nil {
		woc.updateArtifactGCPodInfo(&woc.execWf.Spec.ArtifactGC.ArtifactGC, &podInfo)
		podInfo.podSpecPatch = woc.execWf.Spec.ArtifactGC.PodSpecPatch
	}
	if artifact.ArtifactGC != nil {
		woc.updateArtifactGCPodInfo(artifact.ArtifactGC, &podInfo)
	}
	return podInfo
}

// propagate the information from artifactGC into the podInfo
func (woc *wfOperationCtx) updateArtifactGCPodInfo(artifactGC *wfv1.ArtifactGC, podInfo *podInfo) {

	if artifactGC.ServiceAccountName != "" {
		podInfo.serviceAccount = artifactGC.ServiceAccountName
	}
	if artifactGC.PodMetadata != nil {
		if len(artifactGC.PodMetadata.Labels) > 0 && podInfo.podMetadata.Labels == nil {
			podInfo.podMetadata.Labels = make(map[string]string)
		}
		for labelKey, labelValue := range artifactGC.PodMetadata.Labels {
			podInfo.podMetadata.Labels[labelKey] = labelValue
		}
		if len(artifactGC.PodMetadata.Annotations) > 0 && podInfo.podMetadata.Annotations == nil {
			podInfo.podMetadata.Annotations = make(map[string]string)
		}
		for annotationKey, annotationValue := range artifactGC.PodMetadata.Annotations {
			podInfo.podMetadata.Annotations[annotationKey] = annotationValue
		}
	}

}
