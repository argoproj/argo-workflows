package util

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateHardMemoryQuota(clientset kubernetes.Interface, namespace, name, memoryLimit string) (*corev1.ResourceQuota, error) {
	resourceList := corev1.ResourceList{
		corev1.ResourceLimitsMemory: resource.MustParse(memoryLimit),
	}
	return CreateResourceQuota(clientset, namespace, name, resourceList)
}

func CreateHardStorageQuota(clientset kubernetes.Interface, namespace, name, storageLimit string) (*corev1.ResourceQuota, error) {
	resourceList := corev1.ResourceList{
		"requests.storage": resource.MustParse(storageLimit),
	}
	return CreateResourceQuota(clientset, namespace, name, resourceList)
}

func CreateResourceQuota(clientset kubernetes.Interface, namespace, name string, rl corev1.ResourceList) (*corev1.ResourceQuota, error) {
	return clientset.CoreV1().ResourceQuotas(namespace).Create(&corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"argo-e2e": "true"},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: rl,
		},
	})
}

func DeleteQuota(clientset kubernetes.Interface, quota *corev1.ResourceQuota) error {
	if quota == nil {
		return nil
	}
	return clientset.CoreV1().ResourceQuotas(quota.Namespace).Delete(quota.Name, &metav1.DeleteOptions{})
}
