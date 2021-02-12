package resource

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func DurationForPod(pod *corev1.Pod) wfv1.ResourcesDuration {
	summaries := Summaries{}
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		// Initialize summaries with default limits for CPU and memory.
		summaries[c.Name] = Summary{ResourceList: map[corev1.ResourceName]resource.Quantity{
			// https://medium.com/@betz.mark/understanding-resource-limits-in-kubernetes-cpu-time-9eff74d3161b
			corev1.ResourceCPU: resource.MustParse("100m"),
			// https://medium.com/@betz.mark/understanding-resource-limits-in-kubernetes-memory-6b41e9a955f9
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		}}
		// Update with user-configured resources (falls back to limits as == requests, same as Kubernetes).
		for name, quantity := range c.Resources.Limits {
			summaries[c.Name].ResourceList[name] = quantity
		}
		for name, quantity := range c.Resources.Requests {
			summaries[c.Name].ResourceList[name] = quantity
		}
	}
	for _, c := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		summaries[c.Name] = Summary{ResourceList: summaries[c.Name].ResourceList, ContainerState: c.State}
	}
	return summaries.Duration()
}
