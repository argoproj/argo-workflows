package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/strategicpatch"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/intstr"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/entrypoint"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

var (
	volumeVarArgo = apiv1.Volume{
		Name: "var-run-argo",
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	volumeMountVarArgo = apiv1.VolumeMount{
		Name:      volumeVarArgo.Name,
		MountPath: common.VarRunArgoPath,
	}
	volumeTmpDir = apiv1.Volume{
		Name: "tmp-dir-argo",
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	maxEnvVarLen = 131072
)

// scheduleOnDifferentHost adds affinity to prevent retry on the same host when
// retryStrategy.affinity.nodeAntiAffinity{} is specified
func (woc *wfOperationCtx) scheduleOnDifferentHost(ctx context.Context, node *wfv1.NodeStatus, pod *apiv1.Pod) error {
	if node != nil && pod != nil {
		if retryNode := FindRetryNode(woc.wf.Status.Nodes, node.ID); retryNode != nil {
			// recover template for the retry node
			scope, name := retryNode.GetTemplateScope()
			tmplCtx, err := woc.createTemplateContext(ctx, scope, name)
			if err != nil {
				return err
			}
			_, retryTmpl, _, err := tmplCtx.ResolveTemplate(ctx, retryNode)
			if err != nil {
				return err
			}
			if retryStrategy := woc.retryStrategy(retryTmpl); retryStrategy != nil {
				RetryOnDifferentHost(retryNode.ID)(*retryStrategy, woc.wf.Status.Nodes, pod)
			}
		}
	}
	return nil
}

type createWorkflowPodOpts struct {
	includeScriptOutput bool
	onExitPod           bool
	executionDeadline   time.Time
}

func (woc *wfOperationCtx) processPodSpecPatch(ctx context.Context, tmpl *wfv1.Template, pod *apiv1.Pod) ([]string, error) {
	podSpecPatches := []string{}
	localParams := make(map[string]string)
	if tmpl.IsPodType() {
		localParams[common.LocalVarPodName] = pod.Name
	}
	toProcess := []string{}
	if woc.execWf.Spec.HasPodSpecPatch() {
		toProcess = append(toProcess, woc.execWf.Spec.PodSpecPatch)
	}
	if tmpl.HasPodSpecPatch() {
		toProcess = append(toProcess, tmpl.PodSpecPatch)
	}

	for _, patch := range toProcess {
		newTmpl := tmpl.DeepCopy()
		newTmpl.PodSpecPatch = patch
		processedTmpl, err := common.ProcessArgs(ctx, newTmpl, &wfv1.Arguments{}, woc.globalParams, localParams, false, woc.wf.Namespace, woc.controller.configMapInformer.GetIndexer())
		if err != nil {
			return nil, errors.Wrap(err, "", "Failed to substitute the PodSpecPatch variables")
		}
		podSpecPatches = append(podSpecPatches, processedTmpl.PodSpecPatch)
	}
	return podSpecPatches, nil

}

func (woc *wfOperationCtx) createWorkflowPod(ctx context.Context, nodeName string, mainCtrs []apiv1.Container, tmpl *wfv1.Template, opts *createWorkflowPodOpts) (*apiv1.Pod, error) {
	nodeID := woc.wf.NodeID(nodeName)

	// we must check to see if the pod exists rather than just optimistically creating the pod and see if we get
	// an `AlreadyExists` error because we won't get that error if there is not enough resources.
	// Performance enhancement: Code later in this func is expensive to execute, so return quickly if we can.
	existing, exists, err := woc.podExists(nodeID)
	if err != nil {
		return nil, err
	}

	if exists {
		woc.log.WithFields(logging.Fields{"podPhase": existing.Status.Phase, "nodeName": nodeName, "nodeID": nodeID}).Debug(ctx, "Skipped pod creation: already exists")
		return existing, nil
	}

	if !woc.GetShutdownStrategy().ShouldExecute(opts.onExitPod) {
		// Do not create pods if we are shutting down
		woc.markNodePhase(ctx, nodeName, wfv1.NodeFailed, fmt.Sprintf("workflow shutdown with strategy: %s", woc.GetShutdownStrategy()))
		return nil, nil
	}

	tmpl = tmpl.DeepCopy()
	wfSpec := woc.execWf.Spec.DeepCopy()

	for i, c := range mainCtrs {
		if c.Name == "" || tmpl.GetType() != wfv1.TemplateTypeContainerSet {
			c.Name = common.MainContainerName
		}
		// Allow customization of main container resources.
		if ctrDefaults := woc.controller.Config.MainContainer; ctrDefaults != nil {
			// essentially merge the defaults, then the template, into the container
			a, err := json.Marshal(ctrDefaults)
			if err != nil {
				return nil, err
			}
			b, err := json.Marshal(c)
			if err != nil {
				return nil, err
			}

			mergedContainerByte, err := strategicpatch.StrategicMergePatch(a, b, apiv1.Container{})
			if err != nil {
				return nil, err
			}
			c = apiv1.Container{}
			if err := json.Unmarshal(mergedContainerByte, &c); err != nil {
				return nil, err
			}
		}

		mainCtrs[i] = c
	}

	var activeDeadlineSeconds *int64
	wfDeadline := woc.getWorkflowDeadline()
	tmplActiveDeadlineSeconds, err := intstr.Int64(tmpl.ActiveDeadlineSeconds)
	if err != nil {
		return nil, err
	}
	if wfDeadline == nil || opts.onExitPod { // ignore the workflow deadline for exit handler so they still run if the deadline has passed
		activeDeadlineSeconds = tmplActiveDeadlineSeconds
	} else {
		wfActiveDeadlineSeconds := int64((*wfDeadline).Sub(time.Now().UTC()).Seconds())
		if wfActiveDeadlineSeconds <= 0 {
			return nil, nil
		} else if tmpl.ActiveDeadlineSeconds == nil || wfActiveDeadlineSeconds < *tmplActiveDeadlineSeconds {
			activeDeadlineSeconds = &wfActiveDeadlineSeconds
		} else {
			activeDeadlineSeconds = tmplActiveDeadlineSeconds
		}
	}

	// If the template is marked for debugging no deadline will be set
	for _, c := range mainCtrs {
		for _, env := range c.Env {
			if env.Name == "ARGO_DEBUG_PAUSE_BEFORE" || env.Name == "ARGO_DEBUG_PAUSE_AFTER" {
				activeDeadlineSeconds = nil
			}
		}
	}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GeneratePodName(woc.wf.Name, nodeName, tmpl.Name, nodeID, util.GetWorkflowPodNameVersion(woc.wf)),
			Namespace: woc.wf.Namespace,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name, // Allows filtering by pods related to specific workflow
				common.LabelKeyCompleted: "false",     // Allows filtering by incomplete workflow pods
			},
			Annotations: map[string]string{
				common.AnnotationKeyNodeName: nodeName,
				common.AnnotationKeyNodeID:   nodeID,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy:         apiv1.RestartPolicyNever,
			Volumes:               woc.createVolumes(tmpl),
			ActiveDeadlineSeconds: activeDeadlineSeconds,
			ImagePullSecrets:      woc.execWf.Spec.ImagePullSecrets,
		},
	}

	if os.Getenv(common.EnvVarPodStatusCaptureFinalizer) == "true" {
		pod.Finalizers = append(pod.Finalizers, common.FinalizerPodStatus)
	}

	if opts.onExitPod {
		// This pod is part of an onExit handler, label it so
		pod.Labels[common.LabelKeyOnExit] = "true"
	}

	if woc.execWf.Spec.HostNetwork != nil {
		pod.Spec.HostNetwork = *woc.execWf.Spec.HostNetwork
	}

	woc.addDNSConfig(pod)

	if woc.controller.Config.InstanceID != "" {
		pod.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}

	woc.addArchiveLocation(ctx, tmpl)
	woc.addInputArtifactLocation(ctx, tmpl)
	woc.addOutputArtifactLocation(ctx, tmpl)

	err = woc.setupServiceAccount(ctx, pod, tmpl)
	if err != nil {
		return nil, err
	}

	if (tmpl.GetType() != wfv1.TemplateTypeResource && tmpl.GetType() != wfv1.TemplateTypeData) || (tmpl.GetType() == wfv1.TemplateTypeResource && tmpl.SaveLogsAsArtifact()) {
		// we do not need the wait container for data templates because
		// argoexec runs as the main container and will perform the job of
		// annotating the outputs or errors, making the wait container redundant.
		// for resource template, add a wait container to collect logs.
		waitCtr := woc.newWaitContainer(ctx, tmpl)
		pod.Spec.Containers = append(pod.Spec.Containers, *waitCtr)
	}
	// NOTE: the order of the container list is significant. kubelet will pull, create, and start
	// each container sequentially in the order that they appear in this list. For PNS we want the
	// wait container to start before the main, so that it always has the chance to see the main
	// container's PID and root filesystem.
	pod.Spec.Containers = append(pod.Spec.Containers, mainCtrs...)

	// Configuring default container to be used with commands like "kubectl exec/logs".
	// Select "main" container if it's available. In other case use the last container (can happen when pod created from ContainerSet).
	defaultContainer := pod.Spec.Containers[len(pod.Spec.Containers)-1].Name
	for _, c := range pod.Spec.Containers {
		if c.Name == common.MainContainerName {
			defaultContainer = common.MainContainerName
			break
		}
	}
	pod.Annotations[common.AnnotationKeyDefaultContainer] = defaultContainer

	if podGC := woc.execWf.Spec.PodGC; podGC != nil {
		pod.Annotations[common.AnnotationKeyPodGCStrategy] = fmt.Sprintf("%s/%s", podGC.GetStrategy(), woc.getPodGCDelay(ctx, podGC))
	}

	// Add init container only if it needs input artifacts. This is also true for
	// script templates (which needs to populate the script)
	initCtr := woc.newInitContainer(ctx, tmpl)
	pod.Spec.InitContainers = []apiv1.Container{initCtr}

	woc.addSchedulingConstraints(ctx, pod, wfSpec, tmpl, nodeName)
	woc.addMetadata(pod, tmpl)

	// Set initial progress from pod metadata if exists.
	if x, ok := pod.Annotations[common.AnnotationKeyProgress]; ok {
		if p, ok := wfv1.ParseProgress(x); ok {
			node, err := woc.wf.Status.Nodes.Get(nodeID)
			if err != nil {
				woc.log.WithField("nodeID", nodeID).WithPanic().Error(ctx, "was unable to obtain node")
			}
			node.Progress = p
			woc.wf.Status.Nodes.Set(ctx, nodeID, *node)
		}
	}

	err = addVolumeReferences(pod, woc.volumes, tmpl, woc.wf.Status.PersistentVolumeClaims)
	if err != nil {
		return nil, err
	}

	err = woc.addInputArtifactsVolumes(ctx, pod, tmpl)
	if err != nil {
		return nil, err
	}

	if tmpl.GetType() == wfv1.TemplateTypeScript {
		addScriptStagingVolume(pod)
	}

	// addInitContainers, addSidecars and addOutputArtifactsVolumes should be called after all
	// volumes have been manipulated in the main container since volumeMounts are mirrored
	addInitContainers(ctx, pod, tmpl)
	addSidecars(ctx, pod, tmpl)
	addOutputArtifactsVolumes(ctx, pod, tmpl)

	for i, c := range pod.Spec.InitContainers {
		c.VolumeMounts = append(c.VolumeMounts, volumeMountVarArgo)
		pod.Spec.InitContainers[i] = c
	}

	// simplify template by clearing useless `inputs.parameters` and preserving `inputs.artifacts`.
	// preserving `inputs.artifactLocation` too.
	simplifiedTmpl := tmpl.DeepCopy()
	simplifiedTmpl.Inputs = wfv1.Inputs{
		Artifacts:        simplifiedTmpl.Inputs.Artifacts,
		ArtifactLocation: simplifiedTmpl.Inputs.ArtifactLocation,
	}
	envVarTemplateValue := wfv1.MustMarshallJSON(simplifiedTmpl)

	// Add standard environment variables, making pod spec larger
	envVars := []apiv1.EnvVar{
		{Name: common.EnvVarNodeID, Value: nodeID},
		{Name: common.EnvVarIncludeScriptOutput, Value: strconv.FormatBool(opts.includeScriptOutput)},
		{Name: common.EnvVarDeadline, Value: woc.getDeadline(opts).Format(time.RFC3339)},
	}

	// only set tick durations/EnvVarProgressFile if progress is enabled.
	// The progress is only monitored if the tick durations are >0.
	if woc.controller.progressPatchTickDuration != 0 && woc.controller.progressFileTickDuration != 0 {
		envVars = append(envVars,
			apiv1.EnvVar{
				Name:  common.EnvVarProgressFile,
				Value: common.ArgoProgressPath,
			},
			apiv1.EnvVar{
				Name:  common.EnvVarProgressPatchTickDuration,
				Value: woc.controller.progressPatchTickDuration.String(),
			},
			apiv1.EnvVar{
				Name:  common.EnvVarProgressFileTickDuration,
				Value: woc.controller.progressFileTickDuration.String(),
			},
		)
	}

	for i, c := range pod.Spec.InitContainers {
		c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarContainerName, Value: c.Name})
		c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarTemplate, Value: envVarTemplateValue})
		c.Env = append(c.Env, envVars...)
		pod.Spec.InitContainers[i] = c
	}
	for i, c := range pod.Spec.Containers {
		c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarContainerName, Value: c.Name})
		c.Env = append(c.Env, envVars...)
		pod.Spec.Containers[i] = c
	}

	// Perform one last variable substitution here. Some variables come from the from workflow
	// configmap (e.g. archive location) or volumes attribute, and were not substituted
	// in executeTemplate.
	pod, err = substitutePodParams(ctx, pod, woc.globalParams, tmpl)
	if err != nil {
		return nil, err
	}

	// One final check to verify all variables are resolvable for select fields. We are choosing
	// only to check ArchiveLocation for now, since everything else should have been substituted
	// earlier (i.e. in executeTemplate). But archive location is unique in that the variables
	// are formulated from the configmap. We can expand this to other fields as necessary.
	for _, c := range pod.Spec.InitContainers {
		for _, e := range c.Env {
			if e.Name == common.EnvVarTemplate {
				err = json.Unmarshal([]byte(e.Value), tmpl)
				if err != nil {
					return nil, err
				}
				for _, obj := range []interface{}{tmpl.ArchiveLocation} {
					err = validate.VerifyResolvedVariables(obj)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	// Apply the patch string from workflow and template
	var podSpecPatchs []string
	podSpecPatchs, err = woc.processPodSpecPatch(ctx, tmpl, pod)
	if err != nil {
		return nil, err
	}
	if len(podSpecPatchs) > 0 {
		patchedPodSpec, err := util.ApplyPodSpecPatch(pod.Spec, podSpecPatchs...)
		if err != nil {
			return nil, errors.Wrap(err, "", "Error applying PodSpecPatch")
		}
		pod.Spec = *patchedPodSpec
	}

	for i, c := range pod.Spec.Containers {
		if c.Name != common.WaitContainerName {
			// https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#notes
			if len(c.Command) == 0 {
				x, err := woc.controller.entrypoint.Lookup(ctx, c.Image, entrypoint.Options{
					Namespace: woc.wf.Namespace, ServiceAccountName: woc.execWf.Spec.ServiceAccountName, ImagePullSecrets: woc.execWf.Spec.ImagePullSecrets,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to look-up entrypoint/cmd for image %q, you must either explicitly specify the command, or list the image's command in the index: https://argo-workflows.readthedocs.io/en/latest/workflow-executors/#emissary-emissary: %w", c.Image, err)
				}
				c.Command = x.Entrypoint
				if c.Args == nil { // check nil rather than length, as zero-length is valid args
					c.Args = x.Cmd
				}
			}
			execCmd := append(append([]string{common.VarRunArgoPath + "/argoexec", "emissary"}, woc.getExecutorLogOpts(ctx)...), "--")
			c.Command = append(execCmd, c.Command...)
		}
		if c.Image == woc.controller.executorImage() {
			// mount tmp dir to wait container
			c.VolumeMounts = append(c.VolumeMounts, apiv1.VolumeMount{
				Name:      volumeTmpDir.Name,
				MountPath: "/tmp",
				SubPath:   strconv.Itoa(i),
			})
		}
		c.VolumeMounts = append(c.VolumeMounts, volumeMountVarArgo)
		if x := pod.Spec.TerminationGracePeriodSeconds; x != nil {
			c.Env = append(c.Env, apiv1.EnvVar{Name: common.EnvVarTerminationGracePeriodSeconds, Value: fmt.Sprint(*x)})
		}
		pod.Spec.Containers[i] = c
	}

	offloadEnvVarTemplate := false
	for _, c := range pod.Spec.InitContainers {
		for _, e := range c.Env {
			if e.Name == common.EnvVarTemplate {
				envVarTemplateValue = e.Value
				if len(envVarTemplateValue) > maxEnvVarLen {
					offloadEnvVarTemplate = true
				}
			}
		}
	}

	if offloadEnvVarTemplate {
		cmName := pod.Name
		cm := &apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: woc.wf.Namespace,
				Labels: map[string]string{
					common.LabelKeyWorkflow: woc.wf.Name,
				},
				Annotations: map[string]string{
					common.AnnotationKeyNodeName: nodeName,
					common.AnnotationKeyNodeID:   nodeID,
				},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
				},
			},
			Data: map[string]string{
				common.EnvVarTemplate: envVarTemplateValue,
			},
		}
		created, err := woc.controller.kubeclientset.CoreV1().ConfigMaps(woc.wf.ObjectMeta.Namespace).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			if !apierr.IsAlreadyExists(err) {
				return nil, err
			}
			woc.log.WithField("name", cm.Name).Info(ctx, "Configmap already exists")
		} else {
			woc.log.WithField("name", created.Name).Info(ctx, "Created configmap")
		}

		volumeConfig := apiv1.Volume{
			Name: "argo-env-config",
			VolumeSource: apiv1.VolumeSource{
				ConfigMap: &apiv1.ConfigMapVolumeSource{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: cmName,
					},
				},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, volumeConfig)

		volumeMountConfig := apiv1.VolumeMount{
			Name:      volumeConfig.Name,
			MountPath: common.EnvConfigMountPath,
		}
		for i, c := range pod.Spec.InitContainers {
			for j, e := range c.Env {
				if e.Name == common.EnvVarTemplate {
					e.Value = common.EnvVarTemplateOffloaded
					c.Env[j] = e
				}
			}
			c.VolumeMounts = append(c.VolumeMounts, volumeMountConfig)
			pod.Spec.InitContainers[i] = c
		}
	}

	// Check if the template has exceeded its timeout duration. If it hasn't set the applicable activeDeadlineSeconds
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		woc.log.WithField("nodeName", nodeName).Warn(ctx, "couldn't retrieve node, will get nil templateDeadline")
	}
	templateDeadline, err := woc.checkTemplateTimeout(tmpl, node)
	if err != nil {
		return nil, err
	}

	if err := woc.scheduleOnDifferentHost(ctx, node, pod); err != nil {
		return nil, err
	}

	if templateDeadline != nil && (pod.Spec.ActiveDeadlineSeconds == nil || time.Since(*templateDeadline).Seconds() < float64(*pod.Spec.ActiveDeadlineSeconds)) {
		newActiveDeadlineSeconds := int64(time.Until(*templateDeadline).Seconds())
		if newActiveDeadlineSeconds <= 1 {
			return nil, fmt.Errorf("%s exceeded its deadline", nodeName)
		}
		woc.log.WithFields(logging.Fields{"newActiveDeadlineSeconds": newActiveDeadlineSeconds, "podNamespace": pod.Namespace, "podName": pod.Name}).Debug(ctx, "Setting new activeDeadlineSeconds")
		pod.Spec.ActiveDeadlineSeconds = &newActiveDeadlineSeconds
	}

	if !woc.controller.rateLimiter.Allow() {
		return nil, ErrResourceRateLimitReached
	}

	woc.log.WithFields(logging.Fields{"nodeName": nodeName, "podName": pod.Name}).Debug(ctx, "Creating Pod")

	created, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if the
			// controller fails to persist the workflow after creating the pod.
			woc.log.WithFields(logging.Fields{"nodeName": nodeName, "podName": pod.Name}).Info(ctx, "Failed pod creation: already exists")
			// get a reference to the currently existing Pod since the created pod returned before was nil.
			if existing, err = woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Get(ctx, pod.Name, metav1.GetOptions{}); err == nil {
				return existing, nil
			}
		}
		if errorsutil.IsTransientErr(ctx, err) {
			return nil, err
		}
		woc.log.WithFields(logging.Fields{"nodeName": nodeName, "podName": pod.Name, "error": err}).Info(ctx, "Failed to create pod")
		return nil, errors.InternalWrapError(err)
	}
	woc.log.WithFields(logging.Fields{"nodeName": nodeName, "podName": created.Name}).Info(ctx, "Created pod")
	woc.activePods++
	return created, nil
}

func (woc *wfOperationCtx) podExists(nodeID string) (existing *apiv1.Pod, exists bool, err error) {
	objs, err := woc.controller.PodController.GetPodsByIndex(indexes.NodeIDIndex, woc.wf.Namespace+"/"+nodeID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get pod from informer store: %w", err)
	}

	objectCount := len(objs)

	if objectCount == 0 {
		return nil, false, nil
	}

	if objectCount > 1 {
		return nil, false, fmt.Errorf("expected 1 pod, got %d. This can happen when multiple workflow-controller "+
			"pods are running and both reconciling this Workflow. Check your Argo Workflows installation for a rogue "+
			"workflow-controller. Otherwise, this is a bug", len(objs))
	}

	if existing, ok := objs[0].(*apiv1.Pod); ok {
		return existing, true, nil
	}

	return nil, false, nil
}

func (woc *wfOperationCtx) getDeadline(opts *createWorkflowPodOpts) *time.Time {
	deadline := time.Time{}
	if woc.workflowDeadline != nil && !opts.onExitPod {
		deadline = *woc.workflowDeadline
	}
	if !opts.executionDeadline.IsZero() && (deadline.IsZero() || opts.executionDeadline.Before(deadline)) {
		deadline = opts.executionDeadline
	}
	return &deadline
}

// substitutePodParams returns a pod spec with parameter references substituted as well as pod.name
func substitutePodParams(ctx context.Context, pod *apiv1.Pod, globalParams common.Parameters, tmpl *wfv1.Template) (*apiv1.Pod, error) {
	podParams := globalParams.DeepCopy()
	for _, inParam := range tmpl.Inputs.Parameters {
		podParams["inputs.parameters."+inParam.Name] = inParam.Value.String()
	}
	podParams[common.LocalVarPodName] = pod.Name
	specBytes, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	newSpecBytes, err := template.Replace(ctx, string(specBytes), podParams, true)
	if err != nil {
		return nil, err
	}
	var newSpec apiv1.Pod
	err = json.Unmarshal([]byte(newSpecBytes), &newSpec)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &newSpec, nil
}

func (woc *wfOperationCtx) newInitContainer(ctx context.Context, tmpl *wfv1.Template) apiv1.Container {
	ctr := woc.newExecContainer(common.InitContainerName, tmpl)
	ctr.Command = append([]string{"argoexec", "init"}, woc.getExecutorLogOpts(ctx)...)
	return *ctr
}

func (woc *wfOperationCtx) newWaitContainer(ctx context.Context, tmpl *wfv1.Template) *apiv1.Container {
	ctr := woc.newExecContainer(common.WaitContainerName, tmpl)
	ctr.Command = append([]string{"argoexec", "wait"}, woc.getExecutorLogOpts(ctx)...)
	return ctr
}

func (woc *wfOperationCtx) getExecutorLogOpts(ctx context.Context) []string {
	log := logging.RequireLoggerFromContext(ctx)
	log.WithField("loglevel", string(log.Level())).Info(ctx, "getExecutorLogOpts")
	return []string{"--loglevel", string(log.Level()), "--log-format", woc.controller.executorLogFormat(), "--gloglevel", cmdutil.GetGLogLevel()}
}

func (woc *wfOperationCtx) createEnvVars() []apiv1.EnvVar {
	execEnvVars := []apiv1.EnvVar{
		{
			Name: common.EnvVarPodName,
			ValueFrom: &apiv1.EnvVarSource{
				FieldRef: &apiv1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		},
		{
			Name: common.EnvVarPodUID,
			ValueFrom: &apiv1.EnvVarSource{
				FieldRef: &apiv1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.uid",
				},
			},
		},
		{
			Name:  common.EnvVarWorkflowName,
			Value: woc.wf.Name,
		},
		{
			Name:  common.EnvVarWorkflowUID,
			Value: string(woc.wf.UID),
		},
	}
	if v := woc.controller.Config.InstanceID; v != "" {
		execEnvVars = append(execEnvVars,
			apiv1.EnvVar{Name: common.EnvVarInstanceID, Value: v},
		)
	}
	if woc.controller.Config.Executor != nil {
		execEnvVars = append(execEnvVars, woc.controller.Config.Executor.Env...)
	}
	return execEnvVars
}

func (woc *wfOperationCtx) createVolumes(tmpl *wfv1.Template) []apiv1.Volume {
	var volumes []apiv1.Volume
	if woc.controller.Config.KubeConfig != nil {
		name := woc.controller.Config.KubeConfig.VolumeName
		if name == "" {
			name = common.KubeConfigDefaultVolumeName
		}
		volumes = append(volumes, apiv1.Volume{
			Name: name,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: woc.controller.Config.KubeConfig.SecretName,
				},
			},
		})
	}

	volumes = append(volumes, volumeVarArgo, volumeTmpDir)
	volumes = append(volumes, tmpl.Volumes...)
	return volumes
}

func (woc *wfOperationCtx) newExecContainer(name string, tmpl *wfv1.Template) *apiv1.Container {
	exec := apiv1.Container{
		Name:            name,
		Image:           woc.controller.executorImage(),
		ImagePullPolicy: woc.controller.executorImagePullPolicy(),
		Env:             woc.createEnvVars(),
		Resources:       woc.controller.Config.GetExecutor().Resources,
		SecurityContext: woc.controller.Config.GetExecutor().SecurityContext,
		Args:            woc.controller.Config.GetExecutor().Args,
	}
	// lock down resource pods by default
	if tmpl.GetType() == wfv1.TemplateTypeResource && exec.SecurityContext == nil {
		exec.SecurityContext = common.MinimalCtrSC()
		// TODO: always set RO FS once #10787 is fixed
		exec.SecurityContext.ReadOnlyRootFilesystem = nil
		if exec.Name != common.InitContainerName && exec.Name != common.WaitContainerName {
			exec.SecurityContext.ReadOnlyRootFilesystem = ptr.To(true)
		}
	}
	if woc.controller.Config.KubeConfig != nil {
		path := woc.controller.Config.KubeConfig.MountPath
		if path == "" {
			path = common.KubeConfigDefaultMountPath
		}
		name := woc.controller.Config.KubeConfig.VolumeName
		if name == "" {
			name = common.KubeConfigDefaultVolumeName
		}
		exec.VolumeMounts = append(exec.VolumeMounts, apiv1.VolumeMount{
			Name:      name,
			MountPath: path,
			ReadOnly:  true,
			SubPath:   woc.controller.Config.KubeConfig.SecretKey,
		})
		exec.Args = append(exec.Args, "--kubeconfig="+path)
	}

	executorServiceAccountName := ""
	if tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "" {
		executorServiceAccountName = tmpl.Executor.ServiceAccountName
	} else if woc.execWf.Spec.Executor != nil && woc.execWf.Spec.Executor.ServiceAccountName != "" {
		executorServiceAccountName = woc.execWf.Spec.Executor.ServiceAccountName
	}
	if executorServiceAccountName != "" {
		exec.VolumeMounts = append(exec.VolumeMounts, apiv1.VolumeMount{
			Name:      common.ServiceAccountTokenVolumeName,
			MountPath: common.ServiceAccountTokenMountPath,
			ReadOnly:  true,
		})
	}
	return &exec
}

// addMetadata applies metadata specified in the template
func (woc *wfOperationCtx) addMetadata(pod *apiv1.Pod, tmpl *wfv1.Template) {
	if woc.execWf.Spec.PodMetadata != nil {
		// add workflow-level pod annotations and labels
		for k, v := range woc.execWf.Spec.PodMetadata.Annotations {
			pod.Annotations[k] = v
		}
		for k, v := range woc.execWf.Spec.PodMetadata.Labels {
			pod.Labels[k] = v
		}
	}

	for k, v := range tmpl.Metadata.Annotations {
		pod.Annotations[k] = v
	}
	for k, v := range tmpl.Metadata.Labels {
		pod.Labels[k] = v
	}
}

// addDNSConfig applies DNSConfig to the pod
func (woc *wfOperationCtx) addDNSConfig(pod *apiv1.Pod) {
	if woc.execWf.Spec.DNSPolicy != nil {
		pod.Spec.DNSPolicy = *woc.execWf.Spec.DNSPolicy
	}

	if woc.execWf.Spec.DNSConfig != nil {
		pod.Spec.DNSConfig = woc.execWf.Spec.DNSConfig
	}
}

// addSchedulingConstraints applies any node selectors or affinity rules to the pod, either set in the workflow or the template
func (woc *wfOperationCtx) addSchedulingConstraints(ctx context.Context, pod *apiv1.Pod, wfSpec *wfv1.WorkflowSpec, tmpl *wfv1.Template, nodeName string) {
	// Get boundaryNode Template (if specified)
	boundaryTemplate, err := woc.GetBoundaryTemplate(ctx, nodeName)
	if err != nil {
		woc.log.WithField("nodeName", nodeName).Warn(ctx, "couldn't get boundaryTemplate")
	}
	// Set nodeSelector (if specified)
	if len(tmpl.NodeSelector) > 0 {
		pod.Spec.NodeSelector = tmpl.NodeSelector
	} else if boundaryTemplate != nil && len(boundaryTemplate.NodeSelector) > 0 {
		pod.Spec.NodeSelector = boundaryTemplate.NodeSelector
	} else if len(wfSpec.NodeSelector) > 0 {
		pod.Spec.NodeSelector = wfSpec.NodeSelector
	}
	// Set affinity (if specified)
	if tmpl.Affinity != nil {
		pod.Spec.Affinity = tmpl.Affinity
	} else if boundaryTemplate != nil && boundaryTemplate.Affinity != nil {
		pod.Spec.Affinity = boundaryTemplate.Affinity
	} else if wfSpec.Affinity != nil {
		pod.Spec.Affinity = wfSpec.Affinity
	}
	// Set tolerations (if specified)
	if len(tmpl.Tolerations) > 0 {
		pod.Spec.Tolerations = tmpl.Tolerations
	} else if boundaryTemplate != nil && len(boundaryTemplate.Tolerations) > 0 {
		pod.Spec.Tolerations = boundaryTemplate.Tolerations
	} else if len(wfSpec.Tolerations) > 0 {
		pod.Spec.Tolerations = wfSpec.Tolerations
	}

	// Set scheduler name (if specified)
	if tmpl.SchedulerName != "" {
		pod.Spec.SchedulerName = tmpl.SchedulerName
	} else if wfSpec.SchedulerName != "" {
		pod.Spec.SchedulerName = wfSpec.SchedulerName
	}
	// Set priorityClass (if specified)
	if tmpl.PriorityClassName != "" {
		pod.Spec.PriorityClassName = tmpl.PriorityClassName
	} else if wfSpec.PodPriorityClassName != "" {
		pod.Spec.PriorityClassName = wfSpec.PodPriorityClassName
	}

	// set hostaliases
	pod.Spec.HostAliases = append(pod.Spec.HostAliases, wfSpec.HostAliases...)
	pod.Spec.HostAliases = append(pod.Spec.HostAliases, tmpl.HostAliases...)

	// set pod security context
	if tmpl.SecurityContext != nil {
		pod.Spec.SecurityContext = tmpl.SecurityContext
	} else if wfSpec.SecurityContext != nil {
		pod.Spec.SecurityContext = wfSpec.SecurityContext
	}
}

// GetBoundaryTemplate get a template through the nodeName
func (woc *wfOperationCtx) GetBoundaryTemplate(ctx context.Context, nodeName string) (*wfv1.Template, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		woc.log.WithField("nodeName", nodeName).Warn(ctx, "couldn't retrieve node, will get nil templateDeadline")
		return nil, err
	}
	boundaryTmpl, _, err := woc.GetTemplateByBoundaryID(ctx, node.BoundaryID)
	if err != nil {
		return nil, err
	}
	return boundaryTmpl, nil
}

// GetTemplateByBoundaryID get a template through the node's BoundaryID.
func (woc *wfOperationCtx) GetTemplateByBoundaryID(ctx context.Context, boundaryID string) (*wfv1.Template, bool, error) {
	boundaryNode, err := woc.wf.Status.Nodes.Get(boundaryID)
	if err != nil {
		return nil, false, err
	}
	scope, name := boundaryNode.GetTemplateScope()
	tmplCtx, err := woc.createTemplateContext(ctx, scope, name)
	if err != nil {
		return nil, false, err
	}
	_, boundaryTmpl, templateStored, err := tmplCtx.ResolveTemplate(ctx, boundaryNode)
	if err != nil {
		return nil, templateStored, err
	}
	return boundaryTmpl, templateStored, nil
}

// addVolumeReferences adds any volume mounts that a container/sidecar is referencing, to the pod.spec.volumes
// These are either specified in the workflow.spec.volumes or the workflow.spec.volumeClaimTemplate section
func addVolumeReferences(pod *apiv1.Pod, vols []apiv1.Volume, tmpl *wfv1.Template, pvcs []apiv1.Volume) error {
	switch tmpl.GetType() {
	case wfv1.TemplateTypeContainer, wfv1.TemplateTypeContainerSet, wfv1.TemplateTypeScript, wfv1.TemplateTypeResource, wfv1.TemplateTypeData:
	default:
		return nil
	}

	// getVolByName is a helper to retrieve a volume by its name, either from the volumes or claims section
	getVolByName := func(name string) *apiv1.Volume {
		// Find a volume from template-local volumes.
		for _, vol := range tmpl.Volumes {
			if vol.Name == name {
				return &vol
			}
		}
		// Find a volume from global volumes.
		for _, vol := range vols {
			if vol.Name == name {
				return &vol
			}
		}
		// Find a volume from pvcs.
		for _, pvc := range pvcs {
			if pvc.Name == name {
				return &pvc
			}
		}
		return nil
	}

	addVolumeRef := func(volMounts []apiv1.VolumeMount) error {
		for _, volMnt := range volMounts {
			vol := getVolByName(volMnt.Name)
			if vol == nil {
				return errors.Errorf(errors.CodeBadRequest, "volume '%s' not found in workflow spec", volMnt.Name)
			}
			found := false
			for _, v := range pod.Spec.Volumes {
				if v.Name == vol.Name {
					found = true
					break
				}
			}
			if !found {
				if pod.Spec.Volumes == nil {
					pod.Spec.Volumes = make([]apiv1.Volume, 0)
				}
				pod.Spec.Volumes = append(pod.Spec.Volumes, *vol)
			}
		}
		return nil
	}

	err := addVolumeRef(tmpl.GetVolumeMounts())
	if err != nil {
		return err
	}

	for _, container := range tmpl.InitContainers {
		err := addVolumeRef(container.VolumeMounts)
		if err != nil {
			return err
		}
	}

	for _, sidecar := range tmpl.Sidecars {
		err := addVolumeRef(sidecar.VolumeMounts)
		if err != nil {
			return err
		}
	}

	volumes, volumeMounts := createSecretVolumesAndMounts(tmpl)
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	for idx, container := range pod.Spec.Containers {
		if container.Name == common.WaitContainerName {
			pod.Spec.Containers[idx].VolumeMounts = append(pod.Spec.Containers[idx].VolumeMounts, volumeMounts...)
			break
		}
	}
	for idx, container := range pod.Spec.InitContainers {
		if container.Name == common.InitContainerName {
			pod.Spec.InitContainers[idx].VolumeMounts = append(pod.Spec.InitContainers[idx].VolumeMounts, volumeMounts...)
			break
		}
	}
	if tmpl.Data != nil {
		for idx, container := range pod.Spec.Containers {
			if container.Name == common.MainContainerName {
				pod.Spec.Containers[idx].VolumeMounts = append(pod.Spec.Containers[idx].VolumeMounts, volumeMounts...)
				break
			}
		}
	}

	return nil
}

// addInputArtifactsVolumes sets up the artifacts volume to the pod to support input artifacts to containers.
// In order support input artifacts, the init container shares a emptydir volume with the main container.
// It is the responsibility of the init container to load all artifacts to the mounted emptydir location.
// (e.g. /inputs/artifacts/CODE). The shared emptydir is mapped to the user's desired location in the main
// container.
//
// It is possible that a user specifies overlapping paths of an artifact path with a volume mount,
// (e.g. user wants an external volume mounted at /src, while simultaneously wanting an input artifact
// placed at /src/some/subdirectory). When this occurs, we need to prevent the duplicate bind mounting of
// overlapping volumes, since the outer volume will not see the changes made in the artifact emptydir.
//
// To prevent overlapping bind mounts, both the controller and executor will recognize the overlap between
// the explicit volume mount and the artifact emptydir and prevent all uses of the emptydir for purposes of
// loading data. The controller will omit mounting the emptydir to the artifact path, and the executor
// will load the artifact in the in user's volume (as opposed to the emptydir)
func (woc *wfOperationCtx) addInputArtifactsVolumes(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if len(tmpl.Inputs.Artifacts) == 0 {
		return nil
	}
	artVol := apiv1.Volume{
		Name: "input-artifacts",
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, artVol)

	for i, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			volMount := apiv1.VolumeMount{
				Name:      artVol.Name,
				MountPath: common.ExecutorArtifactBaseDir,
			}
			initCtr.VolumeMounts = append(initCtr.VolumeMounts, volMount)

			// We also add the user supplied mount paths to the init container,
			// in case the executor needs to load artifacts to this volume
			// instead of the artifacts volume
			for _, mnt := range tmpl.GetVolumeMounts() {
				if util.IsWindowsUNCPath(mnt.MountPath, tmpl) {
					continue
				}
				mnt.MountPath = filepath.Join(common.ExecutorMainFilesystemDir, mnt.MountPath)
				initCtr.VolumeMounts = append(initCtr.VolumeMounts, mnt)
			}

			pod.Spec.InitContainers[i] = initCtr
			break
		}
	}

	for i, c := range pod.Spec.Containers {
		if c.Name != common.MainContainerName {
			continue
		}
		for _, art := range tmpl.Inputs.Artifacts {
			err := art.CleanPath()
			if err != nil {
				return errors.Errorf(errors.CodeBadRequest, "error in inputs.artifacts.%s: %s", art.Name, err.Error())
			}
			if !art.HasLocationOrKey() && art.Optional {
				woc.log.WithFields(logging.Fields{"name": art.Name, "path": art.Path}).Info(ctx, "skip volume mount")
				continue
			}
			overlap := common.FindOverlappingVolume(tmpl, art.Path)
			if overlap != nil {
				// artifact path overlaps with a mounted volume. do not mount the
				// artifacts emptydir to the main container. init would have copied
				// the artifact to the user's volume instead
				woc.log.WithFields(logging.Fields{"name": art.Name, "path": art.Path, "overlapName": overlap.Name, "overlapMountPath": overlap.MountPath}).Debug(ctx, "skip volume mount")
				continue
			}
			volMount := apiv1.VolumeMount{
				Name:      artVol.Name,
				MountPath: art.Path,
				SubPath:   art.Name,
			}
			c.VolumeMounts = append(c.VolumeMounts, volMount)
		}
		pod.Spec.Containers[i] = c
	}
	return nil
}

// addOutputArtifactsVolumes mirrors any volume mounts in the main container to the wait sidecar.
// For any output artifacts that were produced in mounted volumes (e.g. PVCs, emptyDirs), the
// wait container will collect the artifacts directly from volumeMount instead of `docker cp`-ing
// them to the wait sidecar. In order for this to work, we mirror all volume mounts in the main
// container under a well-known path.
func addOutputArtifactsVolumes(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) {
	if tmpl.GetType() == wfv1.TemplateTypeResource || tmpl.GetType() == wfv1.TemplateTypeData {
		return
	}

	waitCtrIndex, err := util.FindWaitCtrIndex(pod)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).Info(ctx, "Could not find wait container in pod spec")
		return
	}
	waitCtr := &pod.Spec.Containers[waitCtrIndex]

	for _, c := range pod.Spec.Containers {
		if c.Name != common.MainContainerName {
			continue
		}
		for _, mnt := range c.VolumeMounts {
			if util.IsWindowsUNCPath(mnt.MountPath, tmpl) {
				continue
			}
			mnt.MountPath = filepath.Join(common.ExecutorMainFilesystemDir, mnt.MountPath)
			// ReadOnly is needed to be false for overlapping volume mounts
			mnt.ReadOnly = false
			waitCtr.VolumeMounts = append(waitCtr.VolumeMounts, mnt)
		}
	}
	pod.Spec.Containers[waitCtrIndex] = *waitCtr
}

// addArchiveLocation conditionally updates the template with the default artifact repository
// information configured in the controller, for the purposes of archiving outputs. This is skipped
// for templates which do not need to archive anything, or have explicitly set an archive location
// in the template.
func (woc *wfOperationCtx) addArchiveLocation(ctx context.Context, tmpl *wfv1.Template) {
	if tmpl.ArchiveLocation.HasLocation() {
		// User explicitly set the location. nothing else to do.
		return
	}
	archiveLogs := woc.IsArchiveLogs(tmpl)
	needLocation := archiveLogs
	for _, art := range append(tmpl.Inputs.Artifacts, tmpl.Outputs.Artifacts...) {
		if !art.HasLocation() {
			needLocation = true
		}
	}
	logging.RequireLoggerFromContext(ctx).WithField("needLocation", needLocation).Debug(ctx, "addArchiveLocation")
	if !needLocation {
		return
	}
	tmpl.ArchiveLocation = woc.artifactRepository.ToArtifactLocation()
	tmpl.ArchiveLocation.ArchiveLogs = &archiveLogs
}

func (woc *wfOperationCtx) addInputArtifactLocation(ctx context.Context, tmpl *wfv1.Template) {
	if tmpl.Inputs.ArtifactLocation.HasLocation() {
		return
	}
	if tmpl.Inputs.ArtifactRepositoryRef == nil {
		return
	}
	ref, err := woc.controller.artifactRepositories.Resolve(ctx, tmpl.Inputs.ArtifactRepositoryRef, woc.wf.Namespace)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithField("error", err).Error(ctx, "failed to resolve artifact repository")
		return
	}
	repo, err := woc.controller.artifactRepositories.Get(ctx, ref)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithField("error", err).Error(ctx, "failed to get artifact repository")
		return
	}
	tmpl.Inputs.ArtifactLocation = repo.ToArtifactLocation()
}

func (woc *wfOperationCtx) addOutputArtifactLocation(ctx context.Context, tmpl *wfv1.Template) {
	if tmpl.Outputs.ArtifactLocation.HasLocation() {
		return
	}
	if tmpl.Outputs.ArtifactRepositoryRef == nil {
		return
	}
	ref, err := woc.controller.artifactRepositories.Resolve(ctx, tmpl.Outputs.ArtifactRepositoryRef, woc.wf.Namespace)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithField("error", err).Error(ctx, "failed to resolve artifact repository")
		return
	}
	repo, err := woc.controller.artifactRepositories.Get(ctx, ref)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithField("error", err).Error(ctx, "failed to get artifact repository")
		return
	}
	tmpl.Outputs.ArtifactLocation = repo.ToArtifactLocation()
}

// IsArchiveLogs determines if container should archive logs
// priorities: controller(on) > template > workflow > controller(off)
func (woc *wfOperationCtx) IsArchiveLogs(tmpl *wfv1.Template) bool {
	archiveLogs := woc.artifactRepository.IsArchiveLogs()
	if !archiveLogs {
		if woc.execWf.Spec.ArchiveLogs != nil {
			archiveLogs = *woc.execWf.Spec.ArchiveLogs
		}
		if tmpl.ArchiveLocation != nil && tmpl.ArchiveLocation.ArchiveLogs != nil {
			archiveLogs = *tmpl.ArchiveLocation.ArchiveLogs
		}
	}
	return archiveLogs
}

// setupServiceAccount sets up service account and token.
func (woc *wfOperationCtx) setupServiceAccount(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) error {
	if tmpl.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = tmpl.ServiceAccountName
	} else if woc.execWf.Spec.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = woc.execWf.Spec.ServiceAccountName
	}

	var automountServiceAccountToken *bool
	if tmpl.AutomountServiceAccountToken != nil {
		automountServiceAccountToken = tmpl.AutomountServiceAccountToken
	} else if woc.execWf.Spec.AutomountServiceAccountToken != nil {
		automountServiceAccountToken = woc.execWf.Spec.AutomountServiceAccountToken
	}
	if automountServiceAccountToken != nil && !*automountServiceAccountToken {
		pod.Spec.AutomountServiceAccountToken = automountServiceAccountToken
	}

	executorServiceAccountName := ""
	if tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "" {
		executorServiceAccountName = tmpl.Executor.ServiceAccountName
	} else if woc.execWf.Spec.Executor != nil && woc.execWf.Spec.Executor.ServiceAccountName != "" {
		executorServiceAccountName = woc.execWf.Spec.Executor.ServiceAccountName
	}
	if executorServiceAccountName != "" {
		tokenName, err := woc.getServiceAccountTokenName(ctx, executorServiceAccountName)
		if err != nil {
			return err
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, apiv1.Volume{
			Name: common.ServiceAccountTokenVolumeName,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: tokenName,
				},
			},
		})
	} else if automountServiceAccountToken != nil && !*automountServiceAccountToken {
		return errors.Errorf(errors.CodeBadRequest, "executor.serviceAccountName must not be empty if automountServiceAccountToken is false")
	}
	return nil
}

// addScriptStagingVolume sets up a shared staging volume between the init container
// and main container for the purpose of holding the script source code for script templates
func addScriptStagingVolume(pod *apiv1.Pod) {
	volName := "argo-staging"
	stagingVol := apiv1.Volume{
		Name: volName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, stagingVol)

	for i, initCtr := range pod.Spec.InitContainers {
		if initCtr.Name == common.InitContainerName {
			volMount := apiv1.VolumeMount{
				Name:      volName,
				MountPath: common.ExecutorStagingEmptyDir,
			}
			initCtr.VolumeMounts = append(initCtr.VolumeMounts, volMount)
			pod.Spec.InitContainers[i] = initCtr
			break
		}
	}
	found := false
	for i, ctr := range pod.Spec.Containers {
		if ctr.Name == common.MainContainerName {
			volMount := apiv1.VolumeMount{
				Name:      volName,
				MountPath: common.ExecutorStagingEmptyDir,
			}
			ctr.VolumeMounts = append(ctr.VolumeMounts, volMount)
			pod.Spec.Containers[i] = ctr
			found = true
			break
		}
	}
	if !found {
		panic("Unable to locate main container")
	}
}

// addInitContainers adds all init containers to the pod spec of the step
// Optionally volume mounts from the main container to the init containers
func addInitContainers(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) {
	mainCtr := findMainContainer(pod)
	for _, ctr := range tmpl.InitContainers {
		logging.RequireLoggerFromContext(ctx).WithField("name", ctr.Name).Debug(ctx, "Adding init container")
		if mainCtr != nil && ctr.MirrorVolumeMounts != nil && *ctr.MirrorVolumeMounts {
			mirrorVolumeMounts(ctx, mainCtr, &ctr.Container)
		}
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, ctr.Container)
	}
}

// addSidecars adds all sidecars to the pod spec of the step.
// Optionally volume mounts from the main container to the sidecar
func addSidecars(ctx context.Context, pod *apiv1.Pod, tmpl *wfv1.Template) {
	mainCtr := findMainContainer(pod)
	for _, sidecar := range tmpl.Sidecars {
		logging.RequireLoggerFromContext(ctx).WithField("name", sidecar.Name).Debug(ctx, "Adding sidecar container")
		if mainCtr != nil && sidecar.MirrorVolumeMounts != nil && *sidecar.MirrorVolumeMounts {
			mirrorVolumeMounts(ctx, mainCtr, &sidecar.Container)
		}
		pod.Spec.Containers = append(pod.Spec.Containers, sidecar.Container)
	}
}

// createSecretVolumesAndMounts will retrieve and create Volumes and Volumemount object for Pod
func createSecretVolumesAndMounts(tmpl *wfv1.Template) ([]apiv1.Volume, []apiv1.VolumeMount) {
	allVolumesMap := make(map[string]apiv1.Volume)
	uniqueKeyMap := make(map[string]bool)
	var secretVolumes []apiv1.Volume
	var secretVolMounts []apiv1.VolumeMount

	createArchiveLocationSecret(tmpl, allVolumesMap, uniqueKeyMap)
	createInputArtifactLocationSecret(tmpl, allVolumesMap, uniqueKeyMap)
	createOutputArtifactLocationSecret(tmpl, allVolumesMap, uniqueKeyMap)

	for _, art := range tmpl.Outputs.Artifacts {
		createSecretVolume(allVolumesMap, art, uniqueKeyMap)
	}
	for _, art := range tmpl.Inputs.Artifacts {
		createSecretVolume(allVolumesMap, art, uniqueKeyMap)
	}

	if tmpl.Data != nil {
		if art, needed := tmpl.Data.Source.GetArtifactIfNeeded(); needed {
			createSecretVolume(allVolumesMap, *art, uniqueKeyMap)
		}
	}

	for volMountName, val := range allVolumesMap {
		secretVolumes = append(secretVolumes, val)
		secretVolMounts = append(secretVolMounts, apiv1.VolumeMount{
			Name:      volMountName,
			MountPath: common.SecretVolMountPath + "/" + val.Name,
			ReadOnly:  true,
		})
	}

	return secretVolumes, secretVolMounts
}

func createArchiveLocationSecret(tmpl *wfv1.Template, volMap map[string]apiv1.Volume, uniqueKeyMap map[string]bool) {
	if tmpl.ArchiveLocation == nil {
		return
	}
	createSecretVolumesFromArtifactLocations(volMap, []*wfv1.ArtifactLocation{tmpl.ArchiveLocation}, uniqueKeyMap)
}

func createInputArtifactLocationSecret(tmpl *wfv1.Template, volMap map[string]apiv1.Volume, uniqueKeyMap map[string]bool) {
	if tmpl.Inputs.ArtifactLocation == nil {
		return
	}
	createSecretVolumesFromArtifactLocations(volMap, []*wfv1.ArtifactLocation{tmpl.Inputs.ArtifactLocation}, uniqueKeyMap)
}

func createOutputArtifactLocationSecret(tmpl *wfv1.Template, volMap map[string]apiv1.Volume, uniqueKeyMap map[string]bool) {
	if tmpl.Outputs.ArtifactLocation == nil {
		return
	}
	createSecretVolumesFromArtifactLocations(volMap, []*wfv1.ArtifactLocation{tmpl.Outputs.ArtifactLocation}, uniqueKeyMap)
}

func createSecretVolume(volMap map[string]apiv1.Volume, art wfv1.Artifact, keyMap map[string]bool) {
	createSecretVolumesFromArtifactLocations(volMap, []*wfv1.ArtifactLocation{&art.ArtifactLocation}, keyMap)
}

func createSecretVolumesAndMountsFromArtifactLocations(artifactLocations []*wfv1.ArtifactLocation) ([]apiv1.Volume, []apiv1.VolumeMount) {
	allVolumesMap := make(map[string]apiv1.Volume)
	uniqueKeyMap := make(map[string]bool)
	var secretVolumes []apiv1.Volume
	var secretVolMounts []apiv1.VolumeMount

	createSecretVolumesFromArtifactLocations(allVolumesMap, artifactLocations, uniqueKeyMap)

	for volMountName, val := range allVolumesMap {
		secretVolumes = append(secretVolumes, val)
		secretVolMounts = append(secretVolMounts, apiv1.VolumeMount{
			Name:      volMountName,
			MountPath: common.SecretVolMountPath + "/" + val.Name,
			ReadOnly:  true,
		})
	}

	return secretVolumes, secretVolMounts
}

func createSecretVolumesFromArtifactLocations(volMap map[string]apiv1.Volume, artifactLocations []*wfv1.ArtifactLocation, keyMap map[string]bool) {
	for _, artifactLocation := range artifactLocations {
		if artifactLocation == nil {
			continue
		}
		if artifactLocation.S3 != nil {
			createSecretVal(volMap, artifactLocation.S3.AccessKeySecret, keyMap)
			createSecretVal(volMap, artifactLocation.S3.SecretKeySecret, keyMap)
			if artifactLocation.S3.SessionTokenSecret != nil {
				createSecretVal(volMap, artifactLocation.S3.SessionTokenSecret, keyMap)
			}
			sseCUsed := artifactLocation.S3.EncryptionOptions != nil && artifactLocation.S3.EncryptionOptions.EnableEncryption && artifactLocation.S3.EncryptionOptions.ServerSideCustomerKeySecret != nil
			if sseCUsed {
				createSecretVal(volMap, artifactLocation.S3.EncryptionOptions.ServerSideCustomerKeySecret, keyMap)
			}
			if artifactLocation.S3.CASecret != nil {
				createSecretVal(volMap, artifactLocation.S3.CASecret, keyMap)
			}
		} else if artifactLocation.Git != nil {
			createSecretVal(volMap, artifactLocation.Git.UsernameSecret, keyMap)
			createSecretVal(volMap, artifactLocation.Git.PasswordSecret, keyMap)
			createSecretVal(volMap, artifactLocation.Git.SSHPrivateKeySecret, keyMap)
		} else if artifactLocation.Artifactory != nil {
			createSecretVal(volMap, artifactLocation.Artifactory.UsernameSecret, keyMap)
			createSecretVal(volMap, artifactLocation.Artifactory.PasswordSecret, keyMap)
		} else if artifactLocation.HDFS != nil {
			createSecretVal(volMap, artifactLocation.HDFS.KrbCCacheSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HDFS.KrbKeytabSecret, keyMap)
		} else if artifactLocation.OSS != nil {
			createSecretVal(volMap, artifactLocation.OSS.AccessKeySecret, keyMap)
			createSecretVal(volMap, artifactLocation.OSS.SecretKeySecret, keyMap)
		} else if artifactLocation.GCS != nil {
			createSecretVal(volMap, artifactLocation.GCS.ServiceAccountKeySecret, keyMap)
		} else if artifactLocation.HTTP != nil && artifactLocation.HTTP.Auth != nil {
			createSecretVal(volMap, artifactLocation.HTTP.Auth.BasicAuth.UsernameSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.BasicAuth.PasswordSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.ClientCert.ClientCertSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.ClientCert.ClientKeySecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.OAuth2.ClientIDSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.OAuth2.ClientSecretSecret, keyMap)
			createSecretVal(volMap, artifactLocation.HTTP.Auth.OAuth2.TokenURLSecret, keyMap)
		} else if artifactLocation.Azure != nil {
			createSecretVal(volMap, artifactLocation.Azure.AccountKeySecret, keyMap)
		}
	}
}

func createSecretVal(volMap map[string]apiv1.Volume, secret *apiv1.SecretKeySelector, keyMap map[string]bool) {
	if secret == nil || secret.Name == "" || secret.Key == "" {
		return
	}
	if vol, ok := volMap[secret.Name]; ok {
		key := apiv1.KeyToPath{
			Key:  secret.Key,
			Path: secret.Key,
		}
		if val := keyMap[secret.Name+"-"+secret.Key]; !val {
			keyMap[secret.Name+"-"+secret.Key] = true
			vol.Secret.Items = append(vol.Secret.Items, key)
		}
	} else {
		volume := apiv1.Volume{
			Name: secret.Name,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: secret.Name,
					Items: []apiv1.KeyToPath{
						{
							Key:  secret.Key,
							Path: secret.Key,
						},
					},
				},
			},
		}
		keyMap[secret.Name+"-"+secret.Key] = true
		volMap[secret.Name] = volume
	}
}

// findMainContainer finds main container
func findMainContainer(pod *apiv1.Pod) *apiv1.Container {
	for _, ctr := range pod.Spec.Containers {
		if common.MainContainerName == ctr.Name {
			return &ctr
		}
	}
	return nil
}

// mirrorVolumeMounts mirrors volumeMounts of source container to target container
func mirrorVolumeMounts(ctx context.Context, sourceContainer, targetContainer *apiv1.Container) {
	for _, volMnt := range sourceContainer.VolumeMounts {
		if targetContainer.VolumeMounts == nil {
			targetContainer.VolumeMounts = make([]apiv1.VolumeMount, 0)
		}
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"name": volMnt.Name, "containerName": targetContainer.Name}).Debug(ctx, "Adding volume mount")
		targetContainer.VolumeMounts = append(targetContainer.VolumeMounts, volMnt)

	}
}
