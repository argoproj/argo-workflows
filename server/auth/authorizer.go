package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authUtil "github.com/argoproj/argo-workflows/v3/util/auth"
)

// AccessReview checks if the current context would be allowed to preform an action against the Kubernetes API
// this is used when we aren't going to actually run this verb against the api,
// for example, when querying the workflow archive
func AccessReview(ctx context.Context, namespace, verb, resourceGroup, resourceKind, resourceName string) error {
	kubeClient := GetKubeClient(ctx)
	impersonateClient := GetImpersonateClient(ctx)

	if impersonateClient != nil {
		err := impersonateClient.AccessReview(
			ctx,
			namespace,
			verb,
			resourceGroup,
			resourceKind,
			resourceName,
			"",
		)
		if err != nil {
			return err
		}
	} else {
		allowed, err := authUtil.CanI(ctx, kubeClient, namespace, verb, resourceGroup, resourceKind, resourceName)
		if err != nil {
			return err
		}
		if !allowed {
			// construct a human-friendly string to represent the resource
			resourceString := ""
			if resourceGroup != "" {
				resourceString += resourceGroup
			}
			if resourceKind != "" {
				resourceString += "/" + resourceKind
			}
			if resourceName != "" {
				resourceString += "/" + resourceName
			}
			resourceString = strings.TrimPrefix(resourceString, "/")

			if namespace == "" {
				return status.Errorf(codes.PermissionDenied, "caller is not allowed to '%s' %s in cluster", verb, resourceString)
			} else {
				return status.Errorf(codes.PermissionDenied, "caller is not allowed to '%s' %s in namespace '%s'", verb, resourceString, namespace)
			}
		}
	}

	return nil
}
