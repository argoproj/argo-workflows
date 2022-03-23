package common

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func ClusterWorkflowNamespace(m metav1.Object, primaryCluster string) (namespace, cluster string) {
	return WorkflowNamespace(m), Cluster(m, primaryCluster)
}

func Cluster(m metav1.Object, defaultValue string) string {
	if x, ok := m.GetLabels()[LabelKeyCluster]; ok {
		return x
	}
	return defaultValue
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
