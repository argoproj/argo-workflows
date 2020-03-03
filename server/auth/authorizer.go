package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	authorizationv1 "k8s.io/api/authorization/v1"

	"github.com/argoproj/argo/v2/pkg/apis/workflow"
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

type Authorizer struct {
	ctx    context.Context
	status map[string]authorizationv1.SubjectRulesReviewStatus
}

func (a Authorizer) CanI(verb, resource, namespace, name string) (bool, error) {
	logCtx := log.WithFields(log.Fields{"verb": verb, "resource": resource, "namespace": namespace, "name": name})
	_, ok := a.status[namespace]
	if !ok {
		kubeClientset := GetKubeClient(a.ctx)
		review, err := kubeClientset.AuthorizationV1().SelfSubjectRulesReviews().Create(&authorizationv1.SelfSubjectRulesReview{Spec: authorizationv1.SelfSubjectRulesReviewSpec{Namespace: namespace}})
		if err != nil {
			return false, err
		}
		a.status[namespace] = review.Status
	}
	for _, rule := range a.status[namespace].ResourceRules {
		if allowed(rule.Verbs, verb) &&
			allowed(rule.Resources, resource) &&
			allowed(rule.APIGroups, workflow.Group) &&
			allowed(rule.ResourceNames, name) {
			logCtx.WithFields(log.Fields{"rule": rule, "allowed": true}).Debug("CanI")
			return true, nil
		}
	}
	logCtx.WithField("allowed", false).Debug("CanI")
	return false, nil
}

func NewAuthorizer(ctx context.Context) *Authorizer {
	return &Authorizer{ctx, map[string]authorizationv1.SubjectRulesReviewStatus{}}
}

func allowed(values []string, value string) bool {
	return len(values) == 0 || contains(values, "*") || contains(values, value)
}

func contains(values []string, value string) bool {
	for _, s := range values {
		if value == s {
			return true
		}
	}
	return false
}
