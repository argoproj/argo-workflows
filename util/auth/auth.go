package auth

import (
	log "github.com/sirupsen/logrus"
	auth "k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
)

func CanI(kubeclientset kubernetes.Interface, verb, resource, namespace, name string) (bool, error) {
	logCtx := log.WithFields(log.Fields{"verb": verb, "resource": resource, "namespace": namespace, "name": name})
	logCtx.Debug("CanI")

	review, err := kubeclientset.AuthorizationV1().SelfSubjectAccessReviews().Create(&auth.SelfSubjectAccessReview{
		Spec: auth.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &auth.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     "argoproj.io",
				Resource:  resource,
			},
		},
	})
	if err != nil {
		return false, err
	}
	logCtx.WithField("status", review.Status).Debug("CanI")
	return review.Status.Allowed, nil
}
