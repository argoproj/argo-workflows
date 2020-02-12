package usage

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type containerSummary struct {
	corev1.ResourceList
	corev1.ContainerState
}

func (s containerSummary) duration(now time.Time) time.Duration {
	if s.Terminated != nil {
		return s.Terminated.FinishedAt.Time.Sub(s.Terminated.StartedAt.Time)
	} else if s.Running != nil {
		return now.Sub(s.Running.StartedAt.Time)
	} else {
		return 0
	}
}

func resourceDenominator(r corev1.ResourceName) *resource.Quantity {
	q, ok := map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceCPU:              resource.MustParse("1000m"),
		corev1.ResourceMemory:           resource.MustParse("1Gi"),
		corev1.ResourceStorage:          resource.MustParse("10Gi"),
		corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
	}[r]
	if !ok {
		q = resource.MustParse("1")
	}
	return &q
}

func EstimatePodUsage(pod *corev1.Pod, now time.Time) wfv1.Usage {
	summaries := map[string]containerSummary{}
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		summaries[c.Name] = containerSummary{ResourceList: map[corev1.ResourceName]resource.Quantity{
			// https://medium.com/@betz.mark/understanding-resource-limits-in-kubernetes-cpu-time-9eff74d3161b
			corev1.ResourceCPU: resource.MustParse("100m"),
			// https://medium.com/@betz.mark/understanding-resource-limits-in-kubernetes-memory-6b41e9a955f9
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		}}
		for n, q := range c.Resources.Limits {
			summaries[c.Name].ResourceList[n] = q
		}
		for n, q := range c.Resources.Requests {
			summaries[c.Name].ResourceList[n] = q
		}
	}
	for _, c := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		summaries[c.Name] = containerSummary{ResourceList: summaries[c.Name].ResourceList, ContainerState: c.State}
	}
	usage := wfv1.Usage{}
	for _, s := range summaries {
		duration := s.duration(now)
		for r, q := range s.ResourceList {
			usage = usage.Add(wfv1.Usage{r: wfv1.NewResourceUsage(time.Duration(q.Value() * duration.Nanoseconds() / resourceDenominator(r).Value()))})
		}
	}
	return usage
}
