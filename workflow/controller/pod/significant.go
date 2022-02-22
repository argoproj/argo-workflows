package pod

import (
	"os"

	apiv1 "k8s.io/api/core/v1"
)

func SignificantPodChange(from *apiv1.Pod, to *apiv1.Pod) bool {
	return os.Getenv("ALL_POD_CHANGES_SIGNIFICANT") == "true" ||
		from.Spec.NodeName != to.Spec.NodeName ||
		from.Status.Phase != to.Status.Phase ||
		from.Status.Message != to.Status.Message ||
		from.Status.PodIP != to.Status.PodIP ||
		from.GetDeletionTimestamp() != to.GetDeletionTimestamp() ||
		significantMetadataChange(from.Annotations, to.Annotations) ||
		significantMetadataChange(from.Labels, to.Labels) ||
		significantContainerStatusesChange(from.Status.ContainerStatuses, to.Status.ContainerStatuses) ||
		significantContainerStatusesChange(from.Status.InitContainerStatuses, to.Status.InitContainerStatuses) ||
		significantConditionsChange(from.Status.Conditions, to.Status.Conditions)
}

func significantMetadataChange(from map[string]string, to map[string]string) bool {
	if len(from) != len(to) {
		return true
	}
	for k, v := range from {
		if to[k] != v {
			return true
		}
	}
	// as both annotations must be the same length, the above loop will always catch all changes,
	// we don't need to range with `to`
	return false
}

func significantContainerStatusesChange(from []apiv1.ContainerStatus, to []apiv1.ContainerStatus) bool {
	if len(from) != len(to) {
		return true
	}
	statuses := map[string]apiv1.ContainerStatus{}
	for _, s := range from {
		statuses[s.Name] = s
	}
	for _, s := range to {
		if significantContainerStatusChange(statuses[s.Name], s) {
			return true
		}
	}
	return false
}

func significantContainerStatusChange(from apiv1.ContainerStatus, to apiv1.ContainerStatus) bool {
	return from.Ready != to.Ready || significantContainerStateChange(from.State, to.State)
}

func significantContainerStateChange(from apiv1.ContainerState, to apiv1.ContainerState) bool {
	// waiting has two significant fields and either could potentially change
	return to.Waiting != nil && (from.Waiting == nil || from.Waiting.Message != to.Waiting.Message || from.Waiting.Reason != to.Waiting.Reason) ||
		// running only has one field which is immutable -  so any change is significant
		(to.Running != nil && from.Running == nil) ||
		// I'm assuming this field is immutable - so any change is significant
		(to.Terminated != nil && from.Terminated == nil)
}

func significantConditionsChange(from []apiv1.PodCondition, to []apiv1.PodCondition) bool {
	if len(from) != len(to) {
		return true
	}
	for i, a := range from {
		b := to[i]
		if a.Message != b.Message || a.Reason != b.Reason {
			return true
		}
	}
	return false
}
