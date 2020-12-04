package controller

import (
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo/workflow/common"
)

type podKey = string

func newPodKey(pod *apiv1.Pod) podKey {
	return fmt.Sprintf("%s/%s/%s", clusterNameOrDefault(pod.Labels[common.LabelKeyClusterName]), pod.Namespace, pod.Name)
}

func splitPodKey(key podKey) (clusterName clusterName, namespace string, name string) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]
}
