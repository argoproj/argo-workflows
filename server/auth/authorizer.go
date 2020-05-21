package auth

import (
	"context"

	authUtil "github.com/argoproj/argo/util/auth"
)

func CanI(ctx context.Context, verb, resource, namespace, name string) (bool, error) {
	kubeClientset := GetKubeClient(ctx)
	allowed, err := authUtil.CanI(kubeClientset, verb, "argoproj.io", resource, namespace, name)
	if err != nil {
		return false, err
	}
	return allowed, nil
}
