package controller

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) getAgentPodName() string {
	return woc.wf.NodeID("agent") + "-agent"
}

func (woc *wfOperationCtx) isAgentPod(pod *apiv1.Pod) bool {
	return pod.Name == woc.getAgentPodName()
}

func (woc *wfOperationCtx) reconcileAgentPod(ctx context.Context) error {
	woc.log.Infof("reconcileAgentPod")
	if len(woc.taskSet) == 0 {
		return nil
	}
	pod, err := woc.createAgentPod(ctx)
	if err != nil {
		return err
	}
	// Check Pod is just created
	if pod.Status.Phase != "" {
		woc.updateAgentPodStatus(ctx, pod)
	}
	return nil
}

func (woc *wfOperationCtx) updateAgentPodStatus(ctx context.Context, pod *apiv1.Pod) {
	woc.log.Info("updateAgentPodStatus")
	newPhase, message := assessAgentPodStatus(pod)
	if newPhase == wfv1.WorkflowFailed || newPhase == wfv1.WorkflowError {
		woc.markWorkflowError(ctx, fmt.Errorf("agent pod failed with reason %s", message))
	}
}

func assessAgentPodStatus(pod *apiv1.Pod) (wfv1.WorkflowPhase, string) {
	var newPhase wfv1.WorkflowPhase
	var message string
	log.WithField("namespace", pod.Namespace).
		WithField("podName", pod.Name).
		Info("assessAgentPodStatus")
	switch pod.Status.Phase {
	case apiv1.PodSucceeded, apiv1.PodRunning, apiv1.PodPending:
		return "", ""
	case apiv1.PodFailed:
		newPhase = wfv1.WorkflowFailed
		message = pod.Status.Message
	default:
		newPhase = wfv1.WorkflowError
		message = fmt.Sprintf("Unexpected pod phase for %s: %s", pod.ObjectMeta.Name, pod.Status.Phase)
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
		return nil, nil, fmt.Errorf("failed to check if secret %s exists: %v", name, err)
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
	log := woc.log.WithField("podName", podName)

	obj, exists, err := woc.controller.podInformer.GetStore().Get(cache.ExplicitKey(woc.wf.Namespace + "/" + podName))
	if err != nil {
		return nil, fmt.Errorf("failed to get pod from informer store: %w", err)
	}
	if exists {
		existing, ok := obj.(*apiv1.Pod)
		if ok {
			log.WithField("podPhase", existing.Status.Phase).Debug("Skipped pod creation: already exists")
			return existing, nil
		}
	}

	certVolume, certVolumeMount, err := woc.getCertVolumeMount(ctx, common.CACertificatesVolumeMountName)
	if err != nil {
		return nil, err
	}

	pluginSidecars, pluginVolumes, err := woc.getExecutorPlugins(ctx)
	if err != nil {
		return nil, err
	}

	envVars := []apiv1.EnvVar{
		{Name: common.EnvVarWorkflowName, Value: woc.wf.Name},
		{Name: common.EnvAgentPatchRate, Value: env.LookupEnvStringOr(common.EnvAgentPatchRate, GetRequeueTime().String())},
		{Name: common.EnvVarPluginAddresses, Value: wfv1.MustMarshallJSON(addresses(pluginSidecars))},
		{Name: common.EnvVarPluginNames, Value: wfv1.MustMarshallJSON(names(pluginSidecars))},
	}

	// If the default number of task workers is overridden, then pass it to the agent pod.
	if taskWorkers, exists := os.LookupEnv(common.EnvAgentTaskWorkers); exists {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  common.EnvAgentTaskWorkers,
			Value: taskWorkers,
		})
	}

	serviceAccountName := woc.execWf.Spec.ServiceAccountName
	tokenVolume, tokenVolumeMount, err := woc.getServiceAccountTokenVolume(ctx, serviceAccountName)
	if err != nil {
		return nil, fmt.Errorf("failed to get token volumes: %w", err)
	}

	podVolumes := append(
		pluginVolumes,
		volumeVarArgo,
		*tokenVolume,
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
		SecurityContext: &apiv1.SecurityContext{
			Capabilities: &apiv1.Capabilities{
				Drop: []apiv1.Capability{"ALL"},
			},
			RunAsNonRoot:             pointer.BoolPtr(true),
			RunAsUser:                pointer.Int64Ptr(8737),
			ReadOnlyRootFilesystem:   pointer.BoolPtr(true),
			AllowPrivilegeEscalation: pointer.BoolPtr(false),
		},
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
	agentInitCtr.Args = []string{"agent", "init"}
	// the `main` container runs the actual work
	agentMainCtr := agentCtrTemplate.DeepCopy()
	agentMainCtr.Name = common.MainContainerName
	agentMainCtr.Args = []string{"agent", "main"}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: woc.wf.ObjectMeta.Namespace,
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
			RestartPolicy:    apiv1.RestartPolicyOnFailure,
			ImagePullSecrets: woc.execWf.Spec.ImagePullSecrets,
			SecurityContext: &apiv1.PodSecurityContext{
				RunAsNonRoot: pointer.BoolPtr(true),
				RunAsUser:    pointer.Int64Ptr(8737),
			},
			ServiceAccountName:           serviceAccountName,
			AutomountServiceAccountToken: pointer.BoolPtr(false),
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

	if woc.controller.Config.InstanceID != "" {
		pod.ObjectMeta.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}

	log.Debug("Creating Agent pod")

	created, err := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.ObjectMeta.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		log.WithError(err).Info("Failed to create Agent pod")
		if apierr.IsAlreadyExists(err) {
			return created, nil
		}
		return nil, errors.InternalWrapError(fmt.Errorf("failed to create Agent pod. Reason: %v", err))
	}
	log.Info("Created Agent pod")
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
	var addresses []string
	for _, c := range containers {
		addresses = append(addresses, fmt.Sprintf("http://localhost:%d", c.Ports[0].ContainerPort))
	}
	return addresses
}

func names(containers []apiv1.Container) []string {
	var addresses []string
	for _, c := range containers {
		addresses = append(addresses, c.Name)
	}
	return addresses
}
