package common

import (
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PrimaryCluster() string {
	if v, ok := os.LookupEnv("ARGO_CLUSTER"); ok {
		return v
	}
	return "default"
}

func Cluster(m metav1.Object) string {
	if x, ok := m.GetLabels()[LabelKeyCluster]; ok {
		return x
	}
	return PrimaryCluster()
}

func WorkflowNamespace(m metav1.Object) string {
	if x, ok := m.GetAnnotations()[AnnotationKeyWorkflowNamespace]; ok {
		return x
	}
	return m.GetNamespace()
}

func Namespace(m metav1.Object) string {
	if v, ok := m.GetAnnotations()[AnnotationKeyNamespace]; ok {
		return v
	}
	return m.GetNamespace()
}

func ProfileSecretName(app, cluster string) string {
	return fmt.Sprintf("%s.profile.%s", app, cluster)
}
