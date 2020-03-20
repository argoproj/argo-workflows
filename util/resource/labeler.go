package resource

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func Label(obj metav1.Object, name string, values ...string) {
	for _, value := range values {
		if value == "" {
			continue
		}
		labels := obj.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels[name] = value
		obj.SetLabels(labels)
		return

	}
}
