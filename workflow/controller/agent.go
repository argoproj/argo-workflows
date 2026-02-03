package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"slices"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/env"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func (woc *wfOperationCtx) getAgentPodName() string {
	agentConfig := woc.controller.Config.Agent
	if agentConfig == nil || !agentConfig.RunMultipleWorkflow {
		// Per-workflow agent (current behavior)
		return woc.wf.NodeID("agent") + "-agent"
	}

	// Global agent: one per service account
	serviceAccountName := woc.execWf.Spec.ServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}
	return fmt.Sprintf("argo-agent-%s", serviceAccountName)
}

func (woc *wfOperationCtx) isAgentPod(pod *apiv1.Pod) bool {
	return pod.Name == woc.getAgentPodName()
}

func (woc *wfOperationCtx) reconcileAgentPod(ctx context.Context) error {
	woc.log.Info(ctx, "reconcileAgentPod")
	if len(woc.taskSet) == 0 {
		return nil
	}
	pod, err := woc.createAgentPod(ctx)
	if err != nil {
		return err
	}
	// Check Pod is just created
	if pod != nil && pod.Status.Phase != "" {
		woc.updateAgentPodStatus(ctx, pod)
	}
	return nil
}

func (woc *wfOperationCtx) updateAgentPodStatus(ctx context.Context, pod *apiv1.Pod) {
	woc.log.Info(ctx, "updateAgentPodStatus")
	newPhase, message := assessAgentPodStatus(ctx, pod)
	if newPhase == wfv1.NodeFailed || newPhase == wfv1.NodeError {
		woc.markTaskSetNodesError(ctx, fmt.Errorf(`agent pod failed with reason:"%s"`, message))
	}
}

func assessAgentPodStatus(ctx context.Context, pod *apiv1.Pod) (wfv1.NodePhase, string) {
	var newPhase wfv1.NodePhase
	var message string
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("namespace", pod.Namespace).
		WithField("podName", pod.Name).
		Info(ctx, "assessAgentPodStatus")
	switch pod.Status.Phase {
	case apiv1.PodSucceeded, apiv1.PodRunning, apiv1.PodPending:
		return "", ""
	case apiv1.PodFailed:
		newPhase = wfv1.NodeFailed
		message = pod.Status.Message
	default:
		newPhase = wfv1.NodeError
		message = fmt.Sprintf("Unexpected pod phase for %s: %s", pod.Name, pod.Status.Phase)
	}
	return newPhase, message
}

func (woc *wfOperationCtx) secretExists(ctx context.Context, name string) (bool, error) {
	_, err := woc.controller.kubeclientset.CoreV1().Secrets(woc.wf.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// computeAgentPodSpecHash computes a hash of the agent pod spec
// to detect when plugins or configuration changes
func (woc *wfOperationCtx) computeAgentPodSpecHash(ctx context.Context) (string, error) {
	pluginSidecars, _, err := woc.getExecutorPlugins(ctx)
	if err != nil {
		return "", err
	}

	agentConfig := woc.controller.Config.Agent
	if agentConfig == nil {
		agentConfig = &config.AgentConfig{}
		agentConfig.SetDefaults()
	}

	// Create a stable representation of the spec
	specData := struct {
		ExecutorImage   string
		PluginNames     []string
		PluginImages    []string
		Resources       *apiv1.ResourceRequirements
		SecurityContext *apiv1.SecurityContext
	}{
		ExecutorImage:   woc.controller.executorImage(),
		PluginNames:     make([]string, 0, len(pluginSidecars)),
		PluginImages:    make([]string, 0, len(pluginSidecars)),
		Resources:       agentConfig.Resources,
		SecurityContext: agentConfig.SecurityContext,
	}

	for _, c := range pluginSidecars {
		specData.PluginNames = append(specData.PluginNames, c.Name)
		specData.PluginImages = append(specData.PluginImages, c.Image)
	}

	// Sort for consistency
	sort.Strings(specData.PluginNames)
	sort.Strings(specData.PluginImages)

	specJSON, err := json.Marshal(specData)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(specJSON)
	return hex.EncodeToString(hash[:]), nil
}

func (woc *wfOperationCtx) getCertVolumeMount(ctx context.Context, name string) (*apiv1.Volume, *apiv1.VolumeMount, error) {
	exists, err := woc.secretExists(ctx, name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check if secret %s exists: %w", name, err)
	}
	if exists {
		certVolume := &apiv1.Volume{
			Name: name,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: name,
				},
			}}

		certVolumeMount := &apiv1.VolumeMount{
			Name:      name,
			MountPath: "/etc/ssl/certs/ca-certificates/",
			ReadOnly:  true,
		}

		return certVolume, certVolumeMount, nil
	}
	return nil, nil, nil
}

func (woc *wfOperationCtx) createAgentPod(ctx context.Context) (*apiv1.Pod, error) {
	agentConfig := woc.controller.Config.Agent
	if agentConfig == nil {
		agentConfig = &config.AgentConfig{}
		agentConfig.SetDefaults()
	}

	// Check if controller should create pods
	if !agentConfig.ShouldCreatePod() {
		// External operator manages agent pods, skip creation
		woc.log.Debug(ctx, "Agent pod creation disabled, skipping")
		return nil, nil
	}

	podName := woc.getAgentPodName()
	ctx, log := woc.log.WithField("podName", podName).InContext(ctx)

	// Compute current spec hash
	currentHash, err := woc.computeAgentPodSpecHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compute agent spec hash: %w", err)
	}

	// Check if agent pod already exists
	pod, err := woc.controller.PodController.GetPod(woc.wf.Namespace, podName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod from informer store: %w", err)
	}

	if pod != nil {
		// Agent pod exists - check if spec changed
		existingHash := pod.Annotations[common.AnnotationKeyAgentPodSpecHash]

		if existingHash == currentHash {
			// Spec unchanged, reuse existing pod
			log.Debug(ctx, "Reusing existing agent pod")
			return pod, nil
		}

		// Spec changed (e.g., new plugin added), need to recreate
		log.Info(ctx, "Agent pod spec changed, deleting to recreate")
		err = woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Delete(
			ctx, podName, metav1.DeleteOptions{})
		if err != nil && !apierr.IsNotFound(err) {
			return nil, fmt.Errorf("failed to delete outdated agent pod: %w", err)
		}

		// Requeue workflow for next reconciliation to create new pod
		woc.requeue()
		return nil, nil
	}

	certVolume, certVolumeMount, err := woc.getCertVolumeMount(ctx, common.CACertificatesVolumeMountName)
	if err != nil {
		return nil, err
	}

	pluginSidecars, pluginVolumes, err := woc.getExecutorPlugins(ctx)
	if err != nil {
		return nil, err
	}

	serviceAccountName := woc.execWf.Spec.ServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}

	// Set label selector environment variable for agent
	var labelSelector string
	if agentConfig.RunMultipleWorkflow {
		// Global agent: watch all TaskSets with matching service account
		labelSelector = fmt.Sprintf("%s=%s", common.LabelKeyWorkflowServiceAccount, serviceAccountName)
	} else {
		// Per-workflow agent: watch only this workflow's TaskSet
		labelSelector = fmt.Sprintf("%s=%s", common.LabelKeyWorkflowName, woc.wf.Name)
	}

	envVars := []apiv1.EnvVar{
		{Name: common.EnvVarWorkflowName, Value: woc.wf.Name},
		{Name: common.EnvAgentPatchRate, Value: env.LookupEnvStringOr(common.EnvAgentPatchRate, GetRequeueTime().String())},
		{Name: common.EnvVarPluginAddresses, Value: wfv1.MustMarshallJSON(addresses(pluginSidecars))},
		{Name: common.EnvVarPluginNames, Value: wfv1.MustMarshallJSON(names(pluginSidecars))},
		{Name: common.EnvVarTaskSetLabelSelector, Value: labelSelector},
	}

	// If the default number of task workers is overridden, then pass it to the agent pod.
	if taskWorkers, exists := os.LookupEnv(common.EnvAgentTaskWorkers); exists {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  common.EnvAgentTaskWorkers,
			Value: taskWorkers,
		})
	}
	tokenVolume, tokenVolumeMount, err := woc.getServiceAccountTokenVolume(ctx, serviceAccountName)
	if err != nil {
		return nil, fmt.Errorf("failed to get token volumes: %w", err)
	}

	podVolumes := slices.Concat(
		pluginVolumes,
		[]apiv1.Volume{volumeVarArgo, *tokenVolume},
	)
	podVolumeMounts := []apiv1.VolumeMount{
		volumeMountVarArgo,
		*tokenVolumeMount,
	}
	if certVolume != nil && certVolumeMount != nil {
		podVolumes = append(podVolumes, *certVolume)
		podVolumeMounts = append(podVolumeMounts, *certVolumeMount)
	}
	// Apply configured resources or use defaults
	var agentResources apiv1.ResourceRequirements
	if agentConfig.Resources != nil {
		agentResources = *agentConfig.Resources
	} else {
		agentResources = apiv1.ResourceRequirements{
			Requests: map[apiv1.ResourceName]resource.Quantity{
				"cpu":    resource.MustParse("10m"),
				"memory": resource.MustParse("64M"),
			},
			Limits: map[apiv1.ResourceName]resource.Quantity{
				"cpu":    resource.MustParse(env.LookupEnvStringOr("ARGO_AGENT_CPU_LIMIT", "100m")),
				"memory": resource.MustParse(env.LookupEnvStringOr("ARGO_AGENT_MEMORY_LIMIT", "256M")),
			},
		}
	}

	// Apply configured security context or use defaults
	var agentSecurityContext *apiv1.SecurityContext
	if agentConfig.SecurityContext != nil {
		agentSecurityContext = agentConfig.SecurityContext
	} else {
		agentSecurityContext = common.MinimalCtrSC()
	}

	agentCtrTemplate := apiv1.Container{
		Command:         []string{"argoexec"},
		Image:           woc.controller.executorImage(),
		ImagePullPolicy: woc.controller.executorImagePullPolicy(),
		Env:             envVars,
		SecurityContext: agentSecurityContext,
		Resources:       agentResources,
		VolumeMounts:    podVolumeMounts,
	}
	// the `init` container populates the shared empty-dir volume with tokens
	agentInitCtr := agentCtrTemplate.DeepCopy()
	agentInitCtr.Name = common.InitContainerName
	agentInitCtr.Args = append([]string{"agent", "init"}, woc.getExecutorLogOpts(ctx)...)
	// the `main` container runs the actual work
	agentMainCtr := agentCtrTemplate.DeepCopy()
	agentMainCtr.Name = common.MainContainerName
	agentMainCtr.Args = append([]string{"agent", "main"}, woc.getExecutorLogOpts(ctx)...)

	// Build pod labels
	podLabels := map[string]string{
		common.LabelKeyWorkflow:  woc.wf.Name, // Allows filtering by pods related to specific workflow
		common.LabelKeyCompleted: "false",     // Allows filtering by incomplete workflow pods
		common.LabelKeyComponent: "agent",     // Allows you to identify agent pods and use a different NetworkPolicy on them
	}

	// Add service account label for global agent
	if agentConfig.RunMultipleWorkflow {
		podLabels[common.LabelKeyAgentServiceAccount] = serviceAccountName
	}

	// Build pod annotations
	podAnnotations := map[string]string{
		common.AnnotationKeyDefaultContainer: common.MainContainerName,
		common.AnnotationKeyAgentPodSpecHash: currentHash,
	}

	// Handle owner references based on configuration
	var ownerReferences []metav1.OwnerReference
	if agentConfig.RunMultipleWorkflow {
		// Global agent: remove workflow owner reference (pod outlives workflows)
		ownerReferences = nil
	} else {
		// Per-workflow agent: keep workflow owner reference (current behavior)
		ownerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
		}
	}

	pod = &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            podName,
			Namespace:       woc.wf.Namespace,
			Labels:          podLabels,
			Annotations:     podAnnotations,
			OwnerReferences: ownerReferences,
		},
		Spec: apiv1.PodSpec{
			RestartPolicy:                apiv1.RestartPolicyOnFailure,
			ImagePullSecrets:             woc.execWf.Spec.ImagePullSecrets,
			SecurityContext:              common.MinimalPodSC(),
			ServiceAccountName:           serviceAccountName,
			AutomountServiceAccountToken: ptr.To(false),
			Volumes:                      podVolumes,
			InitContainers: []apiv1.Container{
				*agentInitCtr,
			},
			Containers: append(
				pluginSidecars,
				*agentMainCtr,
			),
		},
	}

	tmpl := &wfv1.Template{}
	woc.addSchedulingConstraints(ctx, pod, woc.execWf.Spec.DeepCopy(), tmpl, "")
	woc.addMetadata(pod, tmpl)
	woc.addDNSConfig(pod)

	if woc.execWf.Spec.HasPodSpecPatch() {
		patchedPodSpec, err := util.ApplyPodSpecPatch(pod.Spec, woc.execWf.Spec.PodSpecPatch)
		if err != nil {
			return nil, errors.Wrap(err, "", "Error applying PodSpecPatch")
		}
		pod.Spec = *patchedPodSpec
	}

	if woc.controller.Config.InstanceID != "" {
		pod.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}

	log.Debug(ctx, "Creating Agent pod")

	created, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		log.WithError(err).Info(ctx, "Failed to create Agent pod")
		if apierr.IsAlreadyExists(err) {
			// get a reference to the currently existing Pod since the created pod returned before was nil.
			if existing, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Get(ctx, pod.Name, metav1.GetOptions{}); err == nil {
				return existing, nil
			}
		}
		if errorsutil.IsTransientErr(ctx, err) {
			woc.requeue()
			return created, nil
		}
		return nil, errors.InternalWrapError(fmt.Errorf("failed to create Agent pod. Reason: %w", err))
	}
	log.Info(ctx, "Created Agent pod")
	return created, nil
}

func (woc *wfOperationCtx) getExecutorPlugins(ctx context.Context) ([]apiv1.Container, []apiv1.Volume, error) {
	var sidecars []apiv1.Container
	var volumes []apiv1.Volume
	namespaces := map[string]bool{} // de-dupes executorPlugins when their namespaces are the same
	namespaces[woc.controller.namespace] = true
	namespaces[woc.wf.Namespace] = true
	for namespace := range namespaces {
		for _, plug := range woc.controller.executorPlugins[namespace] {
			s := plug.Spec.Sidecar
			c := s.Container.DeepCopy()
			c.VolumeMounts = append(c.VolumeMounts, apiv1.VolumeMount{
				Name:      volumeMountVarArgo.Name,
				MountPath: volumeMountVarArgo.MountPath,
				ReadOnly:  true,
				// only mount the token for this plugin, not others
				SubPath: c.Name,
			})
			if s.AutomountServiceAccountToken {
				volume, volumeMount, err := woc.getServiceAccountTokenVolume(ctx, plug.Name+"-executor-plugin")
				if err != nil {
					return nil, nil, err
				}
				volumes = append(volumes, *volume)
				c.VolumeMounts = append(c.VolumeMounts, *volumeMount)
			}
			sidecars = append(sidecars, *c)
		}
	}
	return sidecars, volumes, nil
}

func addresses(containers []apiv1.Container) []string {
	var pluginAddresses []string
	for _, c := range containers {
		pluginAddresses = append(pluginAddresses, fmt.Sprintf("http://localhost:%d", c.Ports[0].ContainerPort))
	}
	return pluginAddresses
}

func names(containers []apiv1.Container) []string {
	var pluginNames []string
	for _, c := range containers {
		pluginNames = append(pluginNames, c.Name)
	}
	return pluginNames
}

// cleanupAgentPodIfUnused checks if agent pod is still needed by other workflows
// and deletes it if not (only when configured to do so)
func (woc *wfOperationCtx) cleanupAgentPod(ctx context.Context) bool {
	woc.log.Info(ctx, "cleanupAgentPod check")
	hasNodes := woc.hasTaskSetNodes()
	agentConfig := woc.controller.Config.Agent
	if agentConfig == nil {
		agentConfig = &config.AgentConfig{}
		agentConfig.SetDefaults()
	}
	shouldDeleteAfterCompletion := agentConfig.ShouldDeleteAfterCompletion()
	createPod := agentConfig.ShouldCreatePod()
	runMultipleWorkflow := agentConfig.RunMultipleWorkflow
	woc.log.WithFields(logging.Fields{
		"hasNodes":                    hasNodes,
		"runMultipleWorkflow":         runMultipleWorkflow,
		"shouldDeleteAfterCompletion": shouldDeleteAfterCompletion,
		"agentPodName":                woc.getAgentPodName(),
		"wfName":                      woc.wf.Name,
	}).Debug(ctx, "cleanupAgentPod decision factors")
	// Check if we should delete agent pod
	if !createPod {
		return false
	} else if !hasNodes {
		return false
	} else if runMultipleWorkflow && woc.hasRunningWorkflowsUsingSameAgentPod(ctx) {
		return false
	} else if runMultipleWorkflow && !shouldDeleteAfterCompletion {
		return false
	}

	return true
}

func (woc *wfOperationCtx) hasRunningWorkflowsUsingSameAgentPod(ctx context.Context) bool {
	serviceAccountName := woc.execWf.Spec.ServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}
	woc.log.WithField("serviceAccountName", serviceAccountName).Info(ctx, "Checking for other workflows using same agent pod")
	// Create selector for incomplete workflows
	selector := labels.Set{
		common.LabelKeyCompleted: "false",
	}.AsSelector()

	// Get all workflows from the informer and filter by namespace and selector
	objs := woc.controller.wfInformer.GetIndexer().List()
	var err error
	for _, obj := range objs {
		un, ok := obj.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		// Filter by namespace
		if un.GetNamespace() != woc.wf.Namespace {
			continue
		}
		// Filter by selector
		if !selector.Matches(labels.Set(un.GetLabels())) {
			continue
		}
		// Convert to Workflow
		var wf wfv1.Workflow
		err = util.FromUnstructuredObj(un, &wf)
		if err != nil {
			continue
		}

		// Skip current workflow
		if wf.Name == woc.wf.Name {
			continue
		}

		// Check if workflow uses same SA
		wfSA := wf.Spec.ServiceAccountName
		if wfSA == "" {
			wfSA = "default"
		}
		if wfSA == serviceAccountName {
			return true
		}

	}

	return false
}
