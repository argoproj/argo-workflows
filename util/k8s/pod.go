package k8s

import (
	"context"
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// GetCurrentPodName returns the name of the current pod using the standard
// Argo Workflows pattern. It first tries to get the pod name from the
// ARGO_POD_NAME environment variable (set via Downward API), and falls back
// to using the Kubernetes client to find the pod by label selector.
func GetCurrentPodName(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string) (string, error) {
	// First try the standard Argo environment variable
	if podName := os.Getenv(common.EnvVarPodName); podName != "" {
		return podName, nil
	}

	// Fallback: use Kubernetes client to find pod by label selector
	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods with selector %s: %w", labelSelector, err)
	}

	if len(podList.Items) == 0 {
		return "", fmt.Errorf("no pods found with selector: %s", labelSelector)
	}

	// Find the first running pod
	for _, pod := range podList.Items {
		if pod.Status.Phase == v1.PodRunning {
			return pod.Name, nil
		}
	}

	// If no running pods, return the first pod found
	return podList.Items[0].Name, nil
}
