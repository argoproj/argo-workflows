package controller

import (
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Should I use "clean-up" or "cleanup"?
// * cleanup is a noun - e.g "The cleanup"
// * clean-up is a verb - e.g. "I clean-up"

type podCleanupKey = string // describes the pod to cleanup + the cleanup action to take
type podCleanupAction = string

const (
	deletePod         podCleanupAction = "deletePod"
	labelPodCompleted podCleanupAction = "labelPodCompleted"
)

func newPodCleanupKey(clusterName wfv1.ClusterName, namespace string, podName string, action podCleanupAction) podCleanupKey {
	return fmt.Sprintf("%s/%s/%s/%v", clusterName, namespace, podName, action)
}

func parsePodCleanupKey(k podCleanupKey) (clusterName wfv1.ClusterName, namespace string, podName string, action podCleanupAction) {
	parts := strings.Split(k, "/")
	if len(parts) != 4 {
		return "", "", "", ""
	}
	return parts[0], parts[1], parts[2], parts[3]

}
