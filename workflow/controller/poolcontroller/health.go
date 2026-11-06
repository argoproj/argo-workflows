package poolcontroller

import (
	"fmt"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/util/podutils"

	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// podHealth is the health of a pool whose Deployment exists. It is degraded
// while none of its pods is ready and its pods can't be created (the
// Deployment's ReplicaFailure condition: quota, admission) or can't start
// their containers (an image that can't be pulled, a missing Secret or
// ConfigMap, a crash loop), and OK otherwise, including while its pods are
// starting. The reason names the problem, not the pod, and leaves out kubelet
// messages that change with every retry, such as pull back-offs', so it
// changes only when the problem does.
func podHealth(deploy *appsv1.Deployment, pods []*apiv1.Pod) dispatch.Health {
	var problems []string
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentReplicaFailure && cond.Status == apiv1.ConditionTrue {
			problems = append(problems, fmt.Sprintf("%s: %s", cond.Reason, cond.Message))
		}
	}
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		if podutils.IsPodReady(pod) {
			return dispatch.Health{State: dispatch.HealthOK}
		}
		for _, status := range slices.Concat(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses) {
			if problem := containerProblem(status); problem != "" {
				problems = append(problems, problem)
			}
		}
	}
	if len(problems) == 0 {
		return dispatch.Health{State: dispatch.HealthOK}
	}
	slices.Sort(problems)
	return dispatch.Health{State: dispatch.HealthDegraded, Reason: problems[0]}
}

// containerProblem describes why a container can't start, or is empty.
func containerProblem(status apiv1.ContainerStatus) string {
	if status.State.Waiting == nil {
		return ""
	}
	switch reason := status.State.Waiting.Reason; reason {
	case "ErrImagePull", "ImagePullBackOff":
		// They alternate while the pull is retried.
		return fmt.Sprintf("ImagePullBackOff: container %q can't pull image %q", status.Name, status.Image)
	case "CrashLoopBackOff":
		return fmt.Sprintf("CrashLoopBackOff: container %q keeps exiting", status.Name)
	case "InvalidImageName", "ErrImageNeverPull", "CreateContainerConfigError", "CreateContainerError":
		return fmt.Sprintf("%s: container %q: %s", reason, status.Name, status.State.Waiting.Message)
	}
	return ""
}

// Degraded is the PoolHealth of a controller that can't run pools: it
// reports every pool degraded, with the reason.
func Degraded(reason string) dispatch.PoolHealth {
	return degraded{State: dispatch.HealthDegraded, Reason: reason}
}

type degraded dispatch.Health

func (d degraded) Health(string, string) (dispatch.Health, bool) {
	return dispatch.Health(d), true
}
