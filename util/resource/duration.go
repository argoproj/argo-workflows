package resource

import (
	"maps"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func DurationForPod(pod *corev1.Pod) wfv1.ResourcesDuration {
	summaries := Summaries{}
	// Pod-level resources are a single budget shared by all containers, so they are
	// attributed once — to the first container that declares no resources of its
	// own — rather than to every such container, which would multiply the one
	// budget by the number of containers.
	podLevelAttributed := false
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		// Initialize summaries with default limits for CPU and memory.
		summaries[c.Name] = Summary{ResourceList: map[corev1.ResourceName]resource.Quantity{
			// https://medium.com/@betz.mark/understanding-resource-limits-in-kubernetes-cpu-time-9eff74d3161b
			corev1.ResourceCPU: resource.MustParse("100m"),
			// https://medium.com/@betz.mark/understanding-resource-limits-in-kubernetes-memory-6b41e9a955f9
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		}}
		if pod.Spec.Resources != nil && !podLevelAttributed && len(c.Resources.Limits)+len(c.Resources.Requests) == 0 {
			maps.Copy(summaries[c.Name].ResourceList, pod.Spec.Resources.Limits)
			maps.Copy(summaries[c.Name].ResourceList, pod.Spec.Resources.Requests)
			podLevelAttributed = true
		}
		// Update with user-configured resources (falls back to limits as == requests, same as Kubernetes).
		maps.Copy(summaries[c.Name].ResourceList, c.Resources.Limits)
		maps.Copy(summaries[c.Name].ResourceList, c.Resources.Requests)
	}
	for _, c := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		summaries[c.Name] = Summary{ResourceList: summaries[c.Name].ResourceList, ContainerState: c.State}
	}
	return summaries.Duration()
}
