package controller

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
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

func (woc *wfOperationCtx) getCertVolumeMount(ctx context.Context, name string) []apiv1.VolumeMount {
	exists, err := woc.secretExists(ctx, name)
	if err != nil {
		woc.log.WithError(err).Errorf("Failed to check if secret %s exists", name)
		return nil
	}
	if exists {
		return []apiv1.VolumeMount{{
			Name:      name,
			MountPath: "/etc/ssl/certs/ca-certificates",
			ReadOnly:  true,
		}}
	}
	return nil
}

func (woc *wfOperationCtx) getCertVolume(ctx context.Context, name string) []apiv1.Volume {
	exists, err := woc.secretExists(ctx, name)
	if err != nil {
		woc.log.WithError(err).Errorf("Failed to check if secret %s exists", name)
		return nil
	}
	if exists {
		return []apiv1.Volume{{
			Name: name,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: name,
				},
			},
		}}
	}
	return nil
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

	pluginSidecars := woc.getExecutorPlugins()
	envVars := []apiv1.EnvVar{
		{Name: common.EnvVarWorkflowName, Value: woc.wf.Name},
		{Name: common.EnvAgentPatchRate, Value: env.LookupEnvStringOr(common.EnvAgentPatchRate, GetRequeueTime().String())},
		{Name: common.EnvVarPluginAddresses, Value: wfv1.MustMarshallJSON(addresses(pluginSidecars))},
	}

	// If the default number of task workers is overridden, then pass it to the agent pod.
	if taskWorkers, exists := os.LookupEnv(common.EnvAgentTaskWorkers); exists {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  common.EnvAgentTaskWorkers,
			Value: taskWorkers,
		})
	}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: woc.wf.ObjectMeta.Namespace,
			Labels: map[string]string{
				common.LabelKeyWorkflow:  woc.wf.Name, // Allows filtering by pods related to specific workflow
				common.LabelKeyCompleted: "false",     // Allows filtering by incomplete workflow pods
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
			},
			Containers: append(
				pluginSidecars,
				apiv1.Container{
					Name:            "main",
					Command:         []string{"argoexec"},
					Args:            []string{"agent"},
					Image:           woc.controller.executorImage(),
					ImagePullPolicy: woc.controller.executorImagePullPolicy(),
					Env:             envVars,

					VolumeMounts: woc.getCertVolumeMount(ctx, common.CACertificatesVolumeMountName),

					SecurityContext: &apiv1.SecurityContext{
						Capabilities: &apiv1.Capabilities{
							Drop: []apiv1.Capability{"ALL"},
						},
						RunAsNonRoot:             pointer.BoolPtr(true),
						RunAsUser:                pointer.Int64Ptr(8737),
						ReadOnlyRootFilesystem:   pointer.BoolPtr(true),
						AllowPrivilegeEscalation: pointer.BoolPtr(false),
					},
				},
			),
			Volumes: woc.getCertVolume(ctx, common.CACertificatesVolumeMountName),
		},
	}

	if woc.controller.Config.InstanceID != "" {
		pod.ObjectMeta.Labels[common.LabelKeyControllerInstanceID] = woc.controller.Config.InstanceID
	}
	if woc.wf.Spec.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = woc.wf.Spec.ServiceAccountName
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

func (woc *wfOperationCtx) getExecutorPlugins() []apiv1.Container {
	var sidecars []apiv1.Container
	namespaces := map[string]bool{} // de-dupes executorPlugins when their namespaces are the same
	namespaces[woc.controller.namespace] = true
	namespaces[woc.wf.Namespace] = true
	for namespace := range namespaces {
		for _, plug := range woc.controller.executorPlugins[namespace] {
			sidecars = append(sidecars, plug.Spec.Sidecar.Container)
		}
	}
	return sidecars
}

func addresses(containers []apiv1.Container) []string {
	var addresses []string
	for _, c := range containers {
		addresses = append(addresses, fmt.Sprintf("http://localhost:%d", c.Ports[0].ContainerPort))
	}
	return addresses
}
