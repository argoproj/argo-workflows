package labels

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/workflow/common"
)

func SetInstanceID(obj metav1.Object, instanceID string) {
	Label(obj, common.LabelKeyControllerInstanceID, instanceID)
}

// label the object with the first non-empty value, if all value are empty, it is not set at all
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
