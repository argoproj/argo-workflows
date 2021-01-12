package controller

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

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

func newPodCleanupKey(clusterName wfv1.ClusterName, gvr schema.GroupVersionResource, namespace string, podName string, action podCleanupAction) podCleanupKey {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%v", clusterName, gvr.Group, gvr.Version, gvr.Resource, namespace, podName, action)
}

func parsePodCleanupKey(k podCleanupKey) (clusterName wfv1.ClusterName, gvr schema.GroupVersionResource, namespace string, podName string, action podCleanupAction) {
	parts := strings.Split(k, "/")
	if len(parts) != 7 {
		return "", schema.GroupVersionResource{}, "", "", ""
	}
	return wfv1.ClusterName(parts[0]), schema.GroupVersionResource{Group: parts[1], Version: parts[2], Resource: parts[3]}, parts[4], parts[5], parts[6]
}
