package auth

import (
	"context"

	authUtil "github.com/argoproj/argo-workflows/v3/util/auth"
)

func CanI(ctx context.Context, verb, resource, namespace string) (bool, error) {
	kubeClientset := GetKubeClient(ctx)
	allowed, err := authUtil.CanI(ctx, kubeClientset, verb, resource, namespace)
	if err != nil {
		return false, err
	}
	return allowed, nil
}
