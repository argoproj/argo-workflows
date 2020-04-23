package instanceid

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/util/labels"
	"github.com/argoproj/argo/workflow/common"
)

func Label(obj metav1.Object, instanceID string) {
	if instanceID != "" {
		labels.Label(obj, common.LabelKeyControllerInstanceID, instanceID)
	} else {
		labels.UnLabel(obj, common.LabelKeyControllerInstanceID)
	}
}

func With(opts metav1.ListOptions, instanceID string) metav1.ListOptions {
	if len(opts.LabelSelector) > 0 {
		opts.LabelSelector += ","
	}
	if instanceID == "" {
		opts.LabelSelector += fmt.Sprintf("!%s", common.LabelKeyControllerInstanceID)
	} else {
		opts.LabelSelector += fmt.Sprintf("%s=%s", common.LabelKeyControllerInstanceID, instanceID)
	}
	return opts
}

func Validate(obj metav1.Object, instanceID string) error {
	l := obj.GetLabels()
	if instanceID == "" {
		if _, ok := l[common.LabelKeyControllerInstanceID]; !ok {
			return nil
		}
	} else if val, ok := l[common.LabelKeyControllerInstanceID]; ok && val == instanceID {
		return nil

	}
	return fmt.Errorf("'%s' is not managed by the current Argo Server", obj.GetName())
}
