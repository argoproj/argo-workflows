package controller

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type informerKey string

func (k informerKey) Split() (clusterName wfv1.ClusterName) {
	parts := strings.Split(string(k), "/")
	if len(parts) != 5 {
		return ""
	}
	return wfv1.ClusterName(parts[0])
}

func joinInformerKey(clusterNamespace wfv1.ClusterNamespaceKey, gvr schema.GroupVersionResource) informerKey {
	clusterName, namespace := clusterNamespace.Split()
	return informerKey(fmt.Sprintf("%s/%s/%s/%s/%s", clusterName, namespace, gvr.Group, gvr.Version, gvr.Resource))
}
