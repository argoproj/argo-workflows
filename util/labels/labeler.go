package labels

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// label the object with value, if the value is empty, it label is deleted.
func Label(obj metav1.Object, name, value string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	if value != "" {
		labels[name] = value
	} else {
		delete(labels, name)
	}
	if len(labels) == 0 {
		obj.SetLabels(nil)
	} else {
		obj.SetLabels(labels)
	}
}
