package instanceid

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/util/labels"
	"github.com/argoproj/argo/util/marker"
	"github.com/argoproj/argo/workflow/common"
)

/*
It might seem that a whole service for instance ID is overkill, but by extending `marker.Service` we
can check at runtime that we actually used instance ID during a request.
*/
type Service interface {
	marker.Service
	Label(ctx context.Context, obj metav1.Object)
	With(ctx context.Context, options *metav1.ListOptions)
	Validate(ctx context.Context, obj metav1.Object) error
	InstanceID(ctx context.Context) string
}

func NewService(instanceId string) Service {
	return &service{marker.NewService(func(fullMethod string) bool {
		return strings.HasPrefix(fullMethod, "/info.InfoService/")
	}), instanceId}
}

type service struct {
	marker.Service
	instanceID string
}

func (s *service) InstanceID(ctx context.Context) string {
	s.Mark(ctx)
	return s.instanceID
}

func (s *service) Label(ctx context.Context, obj metav1.Object) {
	s.Mark(ctx)
	if s.instanceID != "" {
		labels.Label(obj, common.LabelKeyControllerInstanceID, s.instanceID)
	} else {
		labels.UnLabel(obj, common.LabelKeyControllerInstanceID)
	}
}

func (s *service) With(ctx context.Context, opts *metav1.ListOptions) {
	s.Mark(ctx)
	if len(opts.LabelSelector) > 0 {
		opts.LabelSelector += ","
	}
	if s.instanceID == "" {
		opts.LabelSelector += fmt.Sprintf("!%s", common.LabelKeyControllerInstanceID)
	} else {
		opts.LabelSelector += fmt.Sprintf("%s=%s", common.LabelKeyControllerInstanceID, s.instanceID)
	}
}

func (s *service) Validate(ctx context.Context, obj metav1.Object) error {
	s.Mark(ctx)
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
