package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	auth "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CanI(ctx context.Context, kubeclientset kubernetes.Interface, verb, resource, namespace, name string) (bool, error) {
	logCtx := log.WithFields(log.Fields{"verb": verb, "resource": resource, "namespace": namespace, "name": name})
	logCtx.Debug("CanI")

	review, err := kubeclientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, &auth.SelfSubjectAccessReview{
		Spec: auth.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &auth.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     "argoproj.io",
				Resource:  resource,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	logCtx.WithField("status", review.Status).Debug("CanI")
	return review.Status.Allowed, nil
}
