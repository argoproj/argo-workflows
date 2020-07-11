package util

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateConfigMap(clientset kubernetes.Interface, namespace, name string, data map[string]string) (*corev1.ConfigMap, error) {
	return clientset.CoreV1().ConfigMaps(namespace).Create(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"argo-e2e": "true"},
		},
		Data: data,
	})
}

func DeleteConfigMap(clientset kubernetes.Interface, cm *corev1.ConfigMap) error {
	if cm == nil {
		return nil
	}
	return clientset.CoreV1().ConfigMaps(cm.Namespace).Delete(cm.Name, &metav1.DeleteOptions{})
}
