package controller

import (
	"fmt"
	"strings"
)

type podCleanupKey = string // describes the pod to clean-up + the clean-up action to take
type podCleanupAction = string

const (
	deletePod         podCleanupAction = "deletePod"
	labelPodCompleted podCleanupAction = "labelPodCompleted"
)

func joinPodCleanupKey(namespace string, podName string, action podCleanupAction) podCleanupKey {
	return fmt.Sprintf("%s/%s/%v", namespace, podName, action)
}

func splitPodCleanupKey(k podCleanupKey) (namespace string, podName string, action podCleanupAction) {
	parts := strings.Split(k, "/")
	if len(parts) != 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]

}
