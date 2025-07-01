package auth

import (
	"context"

	authUtil "github.com/argoproj/argo-workflows/v3/util/auth"
)

func CanI(ctx context.Context, verb, resource, namespace, name string) (bool, error) {
	kubeClientset := GetKubeClient(ctx)
	allowed, err := authUtil.CanIArgo(ctx, kubeClientset, verb, resource, namespace, name)
	if err != nil {
		return false, err
	}
	return allowed, nil
}
