package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/pkg/plugins/spec"
	"github.com/argoproj/argo-workflows/v4/util/env"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

func (woc *wfOperationCtx) getAgentPodName() string {
	return woc.wf.NodeID("agent") + "-agent"
}

func (woc *wfOperationCtx) getResourceAgentPodName() string {
	return woc.wf.NodeID("resource-agent") + "-resource-agent"
}

func (woc *wfOperationCtx) isAgentPod(pod *apiv1.Pod) bool {
	return pod.Name == woc.getAgentPodName() || pod.Name == woc.getResourceAgentPodName()
}

func (woc *wfOperationCtx) reconcileAgentPod(ctx context.Context) {
	woc.log.Info(ctx, "reconcileAgentPod")
	if len(woc.taskSet) == 0 {
		return
	}
	hasResourceTmpl := false
	hasOtherTmpl := false
	for _, tmpl := range woc.taskSet {
		if tmpl.GetType() == wfv1.TemplateTypeResource {
			hasResourceTmpl = true
		} else {
			hasOtherTmpl = true
		}
	}
	// The two agent pods share one taskset. Reconcile each independently and scope any failure to
	// the nodes it serves, so one pod's failure never fails the other pod's (healthy) nodes.
	if hasOtherTmpl {
		woc.reconcileOneAgentPod(ctx, false)
	}
	if hasResourceTmpl {
		woc.reconcileOneAgentPod(ctx, true)
	}
}

// reconcileOneAgentPod creates (or recovers) a single agent pod and, on a non-transient creation
// failure, marks only the taskset nodes that pod serves as errored — resource-monitor nodes for the
// resource agent, HTTP/plugin nodes for the normal agent — leaving the other pod's nodes untouched.
func (woc *wfOperationCtx) reconcileOneAgentPod(ctx context.Context, resourceAgent bool) {
	var pod *apiv1.Pod
	var err error
	if resourceAgent {
		pod, err = woc.createResourceAgentPod(ctx)
	} else {
		pod, err = woc.createAgentPod(ctx)
	}
	if err != nil {
		woc.markTaskSetNodesError(ctx, fmt.Errorf(`create agent pod failed with reason:"%w"`, err), func(node wfv1.NodeStatus) bool {
			return (node.Type == wfv1.NodeTypeResourceAgent) == resourceAgent
		})
		return
	}
	// Check Pod is just created
	if pod != nil && pod.Status.Phase != "" {
		woc.updateAgentPodStatus(ctx, pod)
	}
}

func (woc *wfOperationCtx) updateAgentPodStatus(ctx context.Context, pod *apiv1.Pod) {
	woc.log.Info(ctx, "updateAgentPodStatus")
	newPhase, message := assessAgentPodStatus(ctx, pod)
	if newPhase == wfv1.NodeFailed || newPhase == wfv1.NodeError {
		// Two agent pods can share the taskset (resource-agent + http/plugin agent). Only fail the
		// nodes served by the pod that actually failed, not every taskset node.
		resourceAgent := pod.Name == woc.getResourceAgentPodName()
		woc.markTaskSetNodesError(ctx, fmt.Errorf(`agent pod failed with reason:"%s"`, message), func(node wfv1.NodeStatus) bool {
			return (node.Type == wfv1.NodeTypeResourceAgent) == resourceAgent
		})
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
	return woc.createTaskSetAgentPod(ctx, false)
}

func (woc *wfOperationCtx) createResourceAgentPod(ctx context.Context) (*apiv1.Pod, error) {
	return woc.createTaskSetAgentPod(ctx, true)
}

func (woc *wfOperationCtx) createTaskSetAgentPod(ctx context.Context, resourceAgent bool) (*apiv1.Pod, error) {
	if woc.controller.Config.DisableAgentPodCreation {
		return nil, nil
	}
	podName := woc.getAgentPodName()
	component := "agent"
	if resourceAgent {
		podName = woc.getResourceAgentPodName()
		component = "resource-agent"
	}
	ctx, log := woc.log.WithField("podName", podName).InContext(ctx)

	pod, err := woc.controller.PodController.GetPod(woc.wf.Namespace, podName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod from informer store: %w", err)
	}
	if pod != nil {
		return pod, nil
	}

	certVolume, certVolumeMount, err := woc.getCertVolumeMount(ctx, common.CACertificatesVolumeMountName)
	if err != nil {
		return nil, err
	}

	var pluginSidecars []apiv1.Container
	var pluginVolumes []apiv1.Volume
	envVars := []apiv1.EnvVar{
		{Name: common.EnvVarWorkflowName, Value: woc.wf.Name},
		{Name: common.EnvVarWorkflowUID, Value: string(woc.wf.UID)},
		{Name: common.EnvAgentPatchRate, Value: env.LookupEnvStringOr(common.EnvAgentPatchRate, GetRequeueTime().String())},
	}
	if !resourceAgent {
		pluginSidecars, pluginVolumes, err = woc.getExecutorPlugins(ctx)
		if err != nil {
			return nil, err
		}
		envVars = append(envVars,
			apiv1.EnvVar{Name: common.EnvVarPluginAddresses, Value: wfv1.MustMarshallJSON(addresses(pluginSidecars))},
			apiv1.EnvVar{Name: common.EnvVarPluginNames, Value: wfv1.MustMarshallJSON(names(pluginSidecars))},
		)
	}

	// If the default number of task workers is overridden, then pass it to the agent pod.
	if taskWorkers, exists := os.LookupEnv(common.EnvAgentTaskWorkers); exists {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  common.EnvAgentTaskWorkers,
			Value: taskWorkers,
		})
	}
	// The resource agent's informer resync period is tunable; pass it through when overridden.
	if resourceAgent {
		if resync, exists := os.LookupEnv(common.EnvAgentResourceInformerResync); exists {
			envVars = append(envVars, apiv1.EnvVar{
				Name:  common.EnvAgentResourceInformerResync,
				Value: resync,
			})
		}
	}

	serviceAccountName := woc.execWf.Spec.ServiceAccountName
	if resourceAgent {
		// The resource agent's informers need list+watch on whole GVRs — broader than
		// workflow pods should carry — so it runs under a dedicated service account,
		// following the `<name>-executor-plugin` convention. Fails if the SA is missing.
		if serviceAccountName == "" {
			serviceAccountName = "default"
		}
		serviceAccountName += "-resource-agent"
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
	agentCtrTemplate := apiv1.Container{
		Command:         []string{"argoexec"},
		Image:           woc.controller.executorImage(),
		ImagePullPolicy: woc.controller.executorImagePullPolicy(),
		Env:             envVars,
		SecurityContext: common.MinimalCtrSC(),
		Resources: apiv1.ResourceRequirements{
			Requests: map[apiv1.ResourceName]resource.Quantity{
				"cpu":    resource.MustParse("10m"),
				"memory": resource.MustParse("64M"),
			},
			Limits: map[apiv1.ResourceName]resource.Quantity{
				"cpu":    resource.MustParse(env.LookupEnvStringOr("ARGO_AGENT_CPU_LIMIT", "100m")),
				"memory": resource.MustParse(env.LookupEnvStringOr("ARGO_AGENT_MEMORY_LIMIT", "256M")),
			},
		},
		VolumeMounts: podVolumeMounts,
	}
	// the `main` container runs the actual work
	agentMainCtr := agentCtrTemplate.DeepCopy()
	agentMainCtr.Name = common.MainContainerName
	agentMainCtr.Args = append([]string{"agent", "main"}, woc.getExecutorLogOpts(ctx)...)
	var initContainers []apiv1.Container
	if resourceAgent {
		agentMainCtr.Args = append([]string{"resource-agent"}, woc.getExecutorLogOpts(ctx)...)
		// resource-agent writes manifests to temp files, but MinimalCtrSC sets a
		// read-only root filesystem; give it a writable /tmp like regular pods have.
		podVolumes = append(podVolumes, volumeTmpDir)
		agentMainCtr.VolumeMounts = append(agentMainCtr.VolumeMounts, apiv1.VolumeMount{
			Name:      volumeTmpDir.Name,
			MountPath: "/tmp",
		})
	} else {
		// the `init` container populates the shared empty-dir volume with plugin tokens
		agentInitCtr := agentCtrTemplate.DeepCopy()
		agentInitCtr.Name = common.InitContainerName
		agentInitCtr.Args = append([]string{"agent", "init"}, woc.getExecutorLogOpts(ctx)...)
		initContainers = append(initContainers, *agentInitCtr)
	}

	pod = &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: woc.wf.Namespace,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name, // Allows filtering by pods related to specific workflow
				common.LabelKeyCompleted: "false",     // Allows filtering by incomplete workflow pods
				common.LabelKeyComponent: component,   // Allows you to identify agent pods and use a different NetworkPolicy on them
			},
			Annotations: map[string]string{
				common.AnnotationKeyDefaultContainer: common.MainContainerName,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(woc.wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy:                apiv1.RestartPolicyOnFailure,
			ImagePullSecrets:             woc.execWf.Spec.ImagePullSecrets,
			SecurityContext:              common.MinimalPodSC(),
			ServiceAccountName:           serviceAccountName,
			AutomountServiceAccountToken: new(false),
			Volumes:                      podVolumes,
			InitContainers:               initContainers,
			Containers: append(
				pluginSidecars,
				*agentMainCtr,
			),
		},
	}

	tmpl := &wfv1.Template{}
	// Agent pods have no boundary node, so there is no boundary template to apply.
	woc.addSchedulingConstraints(ctx, pod, woc.execWf.Spec.DeepCopy(), tmpl, nil)
	woc.addMetadata(pod, tmpl)
	woc.addDNSConfig(pod)

	if woc.execWf.Spec.HasPodSpecPatch() {
		patchedPodSpec, patchErr := util.ApplyPodSpecPatch(pod.Spec, woc.execWf.Spec.PodSpecPatch)
		if patchErr != nil {
			return nil, argoerrors.Wrap(patchErr, "", "Error applying PodSpecPatch")
		}
		pod.Spec = *patchedPodSpec
	}

	if woc.controller.Config.InstanceID != "" {
		pod.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}

	log.Debug(ctx, "Creating Agent pod")

	// Submission goes through the SHARED createPodFromBuild primitive
	// (workflowpod_submit.go), the same impure half the workload path uses. The
	// agent pod carries none of the workload-only machinery (no podLayout, no
	// init-less/legacy split, no node-status side effects), so it wraps the bare
	// pod in a podBuildResult and submits it directly.
	// The fresh flag (genuine create vs AlreadyExists recovery) is irrelevant
	// here: agent pods are not subject to workflow parallelism limits and do
	// not count toward activePods, so it is deliberately discarded.
	created, _, err := woc.createPodFromBuild(ctx, &podBuildResult{Pod: pod}, log)
	if err != nil {
		log.WithError(err).Info(ctx, "Failed to create Agent pod")
		// The agent's transient-error contract: requeue and report no pod,
		// rather than propagating the error up through reconcileAgentPod.
		// createPodFromBuild routes the agent through the shared rate limiter,
		// which returns ErrResourceRateLimitReached when throttled; treat it
		// like the workload path's requeueIfTransientErr condition so a
		// rate-limited agent pod creation requeues gracefully instead of
		// surfacing a hard error.
		if errorsutil.IsTransientErr(ctx, err) || errors.Is(err, ErrResourceRateLimitReached) {
			// The rate limiter runs before Create, so on the rate-limited path
			// the AlreadyExists→Get recovery inside createPodFromBuild never
			// runs. A pod created by a prior reconcile that has not yet reached
			// the informer (checked at the top of this function) must be
			// recovered with a direct Get rather than requeued as though no pod
			// existed.
			if errors.Is(err, ErrResourceRateLimitReached) {
				if existing, getErr := woc.getPod(ctx, podName); getErr == nil {
					log.Info(ctx, "Recovered existing Agent pod on rate-limited create")
					return existing, nil
				}
			}
			woc.requeue()
			return nil, nil
		}
		// createPodFromBuild wraps non-transient failures generically; add the
		// agent-pod context so an agent-pod creation failure is distinguishable
		// from a workload-pod one in logs/status.
		return nil, fmt.Errorf("failed to create Agent pod: %w", err)
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
	wFPlugins, err := woc.execWf.Spec.AsExecutorPluginSpec()
	if err != nil {
		return nil, nil, err
	}
	isGetPluginsFromWorkflow := len(wFPlugins) > 0
	if isGetPluginsFromWorkflow && !woc.controller.enableWorkflowLevelExecutorPlugins {
		return nil, nil, fmt.Errorf(
			"workflow-level executor plugins are disabled in the controller. To enable them, set the environment variable ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS=true",
		)
	}
	if isGetPluginsFromWorkflow {
		for _, plugin := range wFPlugins {
			sidecar, pluginVolume, err := woc.getExecutorPluginComponents(ctx, plugin)
			if err != nil {
				return nil, nil, err
			}
			sidecars = append(sidecars, *sidecar)
			if pluginVolume != nil {
				volumes = append(volumes, *pluginVolume)
			}
		}
	} else {
		for namespace := range namespaces {
			for _, plug := range woc.controller.executorPlugins[namespace] {
				sidecar, pluginVolume, err := woc.getExecutorPluginComponents(ctx, *plug)
				if err != nil {
					return nil, nil, err
				}
				sidecars = append(sidecars, *sidecar)
				if pluginVolume != nil {
					volumes = append(volumes, *pluginVolume)
				}
			}
		}
	}
	return sidecars, volumes, nil
}

func (woc *wfOperationCtx) getExecutorPluginComponents(ctx context.Context, plug spec.Plugin) (*apiv1.Container, *apiv1.Volume, error) {
	s := plug.Spec.Sidecar
	c := s.Container.DeepCopy()
	c.VolumeMounts = append(c.VolumeMounts, apiv1.VolumeMount{
		Name:      volumeMountVarArgo.Name,
		MountPath: volumeMountVarArgo.MountPath,
		ReadOnly:  true,
		// only mount the token for this plugin, not others
		SubPath: c.Name,
	})
	if !s.AutomountServiceAccountToken {
		return c, nil, nil
	}
	volume, volumeMount, err := woc.getServiceAccountTokenVolume(ctx, plug.Name+"-executor-plugin")
	if err != nil {
		return nil, nil, err
	}
	c.VolumeMounts = append(c.VolumeMounts, *volumeMount)
	return c, volume, nil
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
