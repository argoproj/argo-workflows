package rbac

import (
	"context"

	"k8s.io/client-go/kubernetes"

	authutil "github.com/argoproj/argo-workflows/v3/util/auth"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
)

func HasAccessToClusterWorkflowTemplates(ctx context.Context, kubeclientset kubernetes.Interface) bool {
	cwftGetAllowed, err := authutil.CanIArgo(ctx, kubeclientset, "get", "clusterworkflowtemplates", "", "")
	errorsutil.CheckError(err)
	cwftListAllowed, err := authutil.CanIArgo(ctx, kubeclientset, "list", "clusterworkflowtemplates", "", "")
	errorsutil.CheckError(err)
	cwftWatchAllowed, err := authutil.CanIArgo(ctx, kubeclientset, "watch", "clusterworkflowtemplates", "", "")
	errorsutil.CheckError(err)

	if !cwftGetAllowed || !cwftListAllowed || !cwftWatchAllowed {
		return false
	}

	return true
}
