package auth

import (
	"context"

	auth "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// CanIArgo attempts to determine if a verb is actionable by a certain resource, this resource must be an argo resource
func CanIArgo(ctx context.Context, kubeclientset kubernetes.Interface, verb, resource, namespace, name string) (bool, error) {
	logger := logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())
	logger = logger.WithFields(logging.Fields{"verb": verb, "resource": resource, "namespace": namespace, "name": name})
	logger.Debug(ctx, "CanI")
	return CanI(ctx, kubeclientset, []string{verb}, "argoproj.io", namespace, resource)
}

// CanI attempts to determine if a verb is actionable by a certain resource
func CanI(ctx context.Context, kubeclientset kubernetes.Interface, verbs []string, group, namespace, resource string) (bool, error) {
	if len(verbs) == 0 {
		return true, nil
	}
	for _, verb := range verbs {
		review, err := kubeclientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, &auth.SelfSubjectAccessReview{
			Spec: auth.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &auth.ResourceAttributes{
					Namespace: namespace,
					Verb:      verb,
					Group:     group,
					Resource:  resource,
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return false, err
		}
		if !review.Status.Allowed {
			return false, nil
		}
	}
	return true, nil
}
