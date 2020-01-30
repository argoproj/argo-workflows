package cost

import (
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var costDenominators = map[v1.ResourceName]int64{
	v1.ResourceCPU:              1,
	v1.ResourceMemory:           1000000000,
	v1.ResourceStorage:          10000000000,
	v1.ResourceEphemeralStorage: 10000000000,
}

var defaultQuantities = map[v1.ResourceName]resource.Quantity{
	v1.ResourceCPU:              resource.MustParse("1000m"),
	v1.ResourceMemory:           resource.MustParse("1Gi"),
	v1.ResourceStorage:          resource.MustParse("10Gi"),
	v1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
}

type containerSummary struct {
	v1.ResourceList
	v1.ContainerState
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

func EstimatePodsCost(pods []v1.Pod, now time.Time) int64 {
	totalCost := int64(0)
	for _, pod := range pods {
		totalCost += EstimateCost(&pod, now)
	}
	return totalCost
}

func EstimateCost(pod *v1.Pod, now time.Time) int64 {
	totalCost := int64(0)
	summaries := map[string]containerSummary{}
	for _, c := range pod.Spec.Containers {
		summaries[c.Name] = containerSummary{ResourceList: c.Resources.Requests}
	}
	for _, c := range pod.Status.ContainerStatuses {
		summaries[c.Name] = containerSummary{ResourceList: summaries[c.Name].ResourceList, ContainerState: c.State}
	}
	for _, summary := range summaries {
		for name, costDenominator := range costDenominators {
			quantity, ok := summary.ResourceList[name]
			if !ok {
				quantity = defaultQuantities[name]
			}
			value := quantity.Value()
			duration := int64(summary.duration(now).Seconds())
			contribution := value * duration / costDenominator
			log.WithFields(log.Fields{"name": name, "costDenominator": costDenominator, "value": value, "ok": ok,
				"duration": duration, "contribution": contribution}).Info()
			totalCost += contribution
		}
	}

	return totalCost
}
