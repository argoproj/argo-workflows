package usage

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var defaultResourceList = map[corev1.ResourceName]resource.Quantity{
	corev1.ResourceCPU:    resource.MustParse("1000m"),
	corev1.ResourceMemory: resource.MustParse("1Gi"),
}

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

func scale(r corev1.ResourceName) int64 {
	value := scaleValue(r)
	return (&value).Value()
}
func scaleValue(r corev1.ResourceName) resource.Quantity {
	switch r {
	case corev1.ResourceCPU:
		return resource.MustParse("1000m")
	default:
		return resource.MustParse("1Gi")
	}
}

func EstimatePodUsage(pod *corev1.Pod, now time.Time) wfv1.Usage {
	// merge requests and duration into a single list
	summaries := map[string]containerSummary{}
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		summaries[c.Name] = containerSummary{ResourceList: defaultResourceList}
		for name, quantity := range c.Resources.Requests {
			summaries[c.Name].ResourceList[name] = quantity
		}
	}
	for _, c := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		summaries[c.Name] = containerSummary{ResourceList: summaries[c.Name].ResourceList, ContainerState: c.State}
	}
	// then add them all up
	usage := wfv1.Usage{}
	for _, summary := range summaries {
		duration := summary.duration(now)
		for resourceName, list := range summary.ResourceList {
			usage = usage.Add(wfv1.Usage{resourceName: time.Duration(list.Value()/scale(resourceName)) * duration})

		}
	}
	return usage
}
