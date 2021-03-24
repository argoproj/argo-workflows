package instanceid

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/util/labels"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type Service interface {
	Label(obj metav1.Object)
	With(options *metav1.ListOptions)
	Validate(obj metav1.Object) error
	InstanceID() string
}

func NewService(instanceID string) Service {
	return &service{instanceID}
}

type service struct {
	instanceID string
}

func (s *service) InstanceID() string {
	return s.instanceID
}

func (s *service) Label(obj metav1.Object) {
	if s.instanceID != "" {
		labels.Label(obj, common.LabelKeyControllerInstanceID, s.instanceID)
	} else {
		labels.UnLabel(obj, common.LabelKeyControllerInstanceID)
	}
}

func (s *service) With(opts *metav1.ListOptions) {
	if len(opts.LabelSelector) > 0 {
		opts.LabelSelector += ","
	}
	if s.instanceID == "" {
		opts.LabelSelector += fmt.Sprintf("!%s", common.LabelKeyControllerInstanceID)
	} else {
		opts.LabelSelector += fmt.Sprintf("%s=%s", common.LabelKeyControllerInstanceID, s.instanceID)
	}
}

func (s *service) Validate(obj metav1.Object) error {
	l := obj.GetLabels()
	if s.instanceID == "" {
		if _, ok := l[common.LabelKeyControllerInstanceID]; !ok {
			return nil
		}
	} else if val, ok := l[common.LabelKeyControllerInstanceID]; ok && val == s.instanceID {
		return nil
	}
	return fmt.Errorf("'%s' is not managed by the current Argo Server", obj.GetName())
}
