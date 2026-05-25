package controller

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/env"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

func (woc *wfOperationCtx) getAgentPodName() string {
	return woc.wf.NodeID("agent") + "-agent"
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
	podName := woc.getAgentPodName()
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

	pluginSidecars, pluginVolumes, err := woc.getExecutorPlugins(ctx)
	if err != nil {
		return nil, err
	}

	artifactPluginSidecars, artifactPluginVolumes, artifactPluginMounts, artifactPluginNames, err := woc.getAgentArtifactPlugins(ctx)
	if err != nil {
		return nil, err
	}

	envVars := []apiv1.EnvVar{
		{Name: common.EnvVarWorkflowName, Value: woc.wf.Name},
		{Name: common.EnvVarWorkflowUID, Value: string(woc.wf.UID)},
		{Name: common.EnvAgentPatchRate, Value: env.LookupEnvStringOr(common.EnvAgentPatchRate, GetRequeueTime().String())},
		{Name: common.EnvVarPluginAddresses, Value: wfv1.MustMarshallJSON(addresses(pluginSidecars))},
		{Name: common.EnvVarPluginNames, Value: wfv1.MustMarshallJSON(names(pluginSidecars))},
	}
	if len(artifactPluginNames) > 0 {
		// Parity with the wait container — the artifact-plugin driver path
		// keys off this env var to find the plugins available locally.
		envVars = append(envVars, apiv1.EnvVar{
			Name:  common.EnvVarArtifactPluginNames,
			Value: strings.Join(artifactPluginNames, ","),
		})
	}

	// If the default number of task workers is overridden, then pass it to the agent pod.
	if taskWorkers, exists := os.LookupEnv(common.EnvAgentTaskWorkers); exists {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  common.EnvAgentTaskWorkers,
			Value: taskWorkers,
		})
	}

	// The agent pod runs argo machinery (HTTP/plugin/resource template
	// orchestration), not user code, so it inherits the executor SA when one
	// is set — that's the SA designated for argo's own work. This matches the
	// pre-PR resource-template path, where the executor sidecar (running with
	// executor.serviceAccountName) did the kubectl create and polling. Falls
	// back to the workflow SA to preserve existing behavior for workflows
	// that don't configure executor.serviceAccountName.
	serviceAccountName := woc.execWf.Spec.ServiceAccountName
	if woc.execWf.Spec.Executor != nil && woc.execWf.Spec.Executor.ServiceAccountName != "" {
		serviceAccountName = woc.execWf.Spec.Executor.ServiceAccountName
	}
	tokenVolume, tokenVolumeMount, err := woc.getServiceAccountTokenVolume(ctx, serviceAccountName)
	if err != nil {
		return nil, fmt.Errorf("failed to get token volumes: %w", err)
	}

	// Artifact-plugin sidecars run with the pod's AutomountServiceAccountToken
	// disabled, so the standard projection at /var/run/secrets/kubernetes.io/serviceaccount
	// is absent. Plugin RPC handlers that build an in-cluster client to read
	// credential Secrets (e.g. minio accessKey/secretKey) fail without a token
	// at the conventional path. Mount the same SA-bound token volume the agent
	// main container uses — the Volume is already in podVolumes below.
	for i := range artifactPluginSidecars {
		artifactPluginSidecars[i].VolumeMounts = append(artifactPluginSidecars[i].VolumeMounts, *tokenVolumeMount)
	}

	// The agent container runs with ReadOnlyRootFilesystem=true (see
	// common.MinimalCtrSC), so /tmp on the root fs is read-only. Resource
	// templates need scratch space for kubectl manifest files and downloaded
	// manifestFrom artifacts. Shadow /tmp with a tmpfs emptyDir so os.WriteFile
	// / os.CreateTemp / os.MkdirTemp continue to work without code changes.
	agentTmpSize := resource.MustParse("64Mi")
	volumeAgentTmp := apiv1.Volume{
		Name: "tmp",
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{
				Medium:    apiv1.StorageMediumMemory,
				SizeLimit: &agentTmpSize,
			},
		},
	}
	volumeMountAgentTmp := apiv1.VolumeMount{
		Name:      volumeAgentTmp.Name,
		MountPath: "/tmp",
	}

	podVolumes := slices.Concat(
		pluginVolumes,
		artifactPluginVolumes,
		[]apiv1.Volume{volumeVarArgo, volumeAgentTmp, *tokenVolume},
	)
	podVolumeMounts := []apiv1.VolumeMount{
		volumeMountVarArgo,
		volumeMountAgentTmp,
		*tokenVolumeMount,
	}
	// Mount each artifact plugin's socket dir on the agent main container so
	// the in-process artifact driver (plugin.NewDriver) can dial the sidecar
	// over its unix socket — the same contract the wait container relies on.
	podVolumeMounts = append(podVolumeMounts, artifactPluginMounts...)
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
	// the `init` container populates the shared empty-dir volume with tokens
	agentInitCtr := agentCtrTemplate.DeepCopy()
	agentInitCtr.Name = common.InitContainerName
	agentInitCtr.Args = append([]string{"agent", "init"}, woc.getExecutorLogOpts(ctx)...)
	// the `main` container runs the actual work
	agentMainCtr := agentCtrTemplate.DeepCopy()
	agentMainCtr.Name = common.MainContainerName
	agentMainCtr.Args = append([]string{"agent", "main"}, woc.getExecutorLogOpts(ctx)...)

	pod = &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: woc.wf.Namespace,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name, // Allows filtering by pods related to specific workflow
				common.LabelKeyCompleted: "false",     // Allows filtering by incomplete workflow pods
				common.LabelKeyComponent: "agent",     // Allows you to identify agent pods and use a different NetworkPolicy on them
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
			InitContainers: []apiv1.Container{
				*agentInitCtr,
			},
			Containers: slices.Concat(
				pluginSidecars,
				artifactPluginSidecars,
				[]apiv1.Container{*agentMainCtr},
			),
		},
	}

	tmpl := &wfv1.Template{}
	woc.addSchedulingConstraints(ctx, pod, woc.execWf.Spec.DeepCopy(), tmpl, "")
	woc.addMetadata(pod, tmpl)
	woc.addDNSConfig(pod)

	if woc.execWf.Spec.HasPodSpecPatch() {
		patchedPodSpec, patchErr := util.ApplyPodSpecPatch(pod.Spec, woc.execWf.Spec.PodSpecPatch)
		if patchErr != nil {
			return nil, errors.Wrap(patchErr, "", "Error applying PodSpecPatch")
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
			if existing, getErr := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Get(ctx, pod.Name, metav1.GetOptions{}); getErr == nil {
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

// getAgentArtifactPlugins collects the artifact plugin sidecars the agent pod
// must run so resource templates can save their main-logs (and resolve any
// manifestFrom artifacts) via plugin-backed artifact repositories.
//
// Without these sidecars the agent's call to artifacts.NewDriver would block
// inside plugin.NewDriver polling for a unix socket that never appears,
// silently stranding every resource template whose archive location resolves
// to a plugin.
//
// The agent pod is created once per workflow and may handle many resource
// templates over its lifetime, so we union plugin needs across all of them
// rather than per-template. The result mirrors workflowpod.addArtifactPlugins:
// one sidecar per driver, a shared emptyDir per driver for the socket, and a
// volume mount on the agent main container pointing at that socket dir.
func (woc *wfOperationCtx) getAgentArtifactPlugins(ctx context.Context) ([]apiv1.Container, []apiv1.Volume, []apiv1.VolumeMount, []string, error) {
	pluginNameSet := map[wfv1.ArtifactPluginName]bool{}
	for i := range woc.execWf.Spec.Templates {
		tmpl := &woc.execWf.Spec.Templates[i]
		if tmpl.Resource == nil {
			continue
		}
		for _, name := range tmpl.Outputs.Artifacts.GetPluginNames(ctx, woc.artifactRepository, wfv1.IncludeLogs, tmpl.ArchiveLocation) {
			pluginNameSet[name] = true
		}
	}
	if len(pluginNameSet) == 0 {
		return nil, nil, nil, nil, nil
	}

	pluginNames := make([]wfv1.ArtifactPluginName, 0, len(pluginNameSet))
	for n := range pluginNameSet {
		pluginNames = append(pluginNames, n)
	}
	drivers, err := woc.controller.Config.GetArtifactDrivers(pluginNames)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("get artifact drivers for agent: %w", err)
	}

	// Synthetic template so artifactSidecarContainer's TemplateType check
	// (resource → MinimalCtrSC) doesn't misclassify a plugin sidecar.
	syntheticTmpl := &wfv1.Template{}
	sidecars := make([]apiv1.Container, 0, len(drivers))
	volumes := make([]apiv1.Volume, 0, len(drivers))
	mounts := make([]apiv1.VolumeMount, 0, len(drivers))
	sidecarNames := make([]string, 0, len(drivers))
	for _, driver := range drivers {
		ctr, err := woc.artifactSidecarContainer(ctx, syntheticTmpl, driver)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("build artifact sidecar for plugin %q: %w", driver.Name, err)
		}
		// The sidecar's command is /var/run/argo/argoexec — workflow pods get
		// the var-run-argo mount applied in the createWorkflowPod loop, but
		// the agent pod has no such loop. Mount it explicitly so the binary
		// (staged by the agent init container) is visible to the sidecar.
		ctr.VolumeMounts = append(ctr.VolumeMounts, volumeMountVarArgo)
		sidecars = append(sidecars, *ctr)
		volumes = append(volumes, driver.Name.Volume())
		mounts = append(mounts, driver.Name.VolumeMount())
		sidecarNames = append(sidecarNames, ctr.Name)
	}
	return sidecars, volumes, mounts, sidecarNames, nil
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
