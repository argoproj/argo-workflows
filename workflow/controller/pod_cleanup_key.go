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

func newPodCleanupKey(clusterName wfv1.ClusterName, namespace, name string, gvr schema.GroupVersionResource, action podCleanupAction) podCleanupKey {
	return fmt.Sprintf("%s/%s/%s.%s.%s/%s/%v", clusterName, namespace, gvr.Resource, gvr.Version, gvr.Group, name, action)
}

func parsePodCleanupKey(k podCleanupKey) (clusterName wfv1.ClusterName, namespace, name string, gvr schema.GroupVersionResource, action podCleanupAction) {
	parts := strings.Split(k, "/")
	if len(parts) != 5 {
		return "", "", "", schema.GroupVersionResource{}, ""
	}
	gvr1, _ := schema.ParseResourceArg(parts[3])
	if gvr.Empty() {
		return "", "", "", schema.GroupVersionResource{}, ""
	}
	return wfv1.ClusterName(parts[0]), parts[4], parts[5], *gvr1, parts[6]
}
