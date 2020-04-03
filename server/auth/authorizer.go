package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func CanI(ctx context.Context, verb, resource, namespace, name string) (bool, error) {
	kubeClientset := GetKubeClient(ctx)
	logCtx := log.WithFields(log.Fields{"verb": verb, "resource": resource, "namespace": namespace, "name": name})
	logCtx.Debug("CanI")
	review, err := kubeClientset.AuthorizationV1().SelfSubjectAccessReviews().Create(&authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     "argoproj.io",
				Resource:  resource,
				Name:      name,
			},
		},
	})
	if err != nil {
		return false, err
	}
	logCtx.WithField("status", review.Status).Debug("CanI")
	return review.Status.Allowed, nil
}
