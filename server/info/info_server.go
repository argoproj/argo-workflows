package info

import (
	"context"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
)

type infoServer struct {
	managedNamespace string
	links            []*wfv1.Link
}

func (i *infoServer) GetUserInfo(ctx context.Context, _ *infopkg.GetUserInfoRequest) (*infopkg.GetUserInfoResponse, error) {
	claims := auth.GetClaims(ctx)
	if claims != nil {
		return &infopkg.GetUserInfoResponse{
			Subject:            claims.Subject,
			Issuer:             claims.Issuer,
			Groups:             claims.Groups,
			Email:              claims.Email,
			EmailVerified:      claims.EmailVerified,
			ServiceAccountName: claims.ServiceAccountName,
		}, nil
	}
	return &infopkg.GetUserInfoResponse{}, nil
}

func (i *infoServer) GetInfo(context.Context, *infopkg.GetInfoRequest) (*infopkg.InfoResponse, error) {
	return &infopkg.InfoResponse{ManagedNamespace: i.managedNamespace, Links: i.links}, nil
}

func (i *infoServer) GetVersion(context.Context, *infopkg.GetVersionRequest) (*wfv1.Version, error) {
	version := argo.GetVersion()
	return &version, nil
}

func (i *infoServer) ListNamespaces(ctx context.Context, _ *infopkg.ListNamespacesRequest) (*infopkg.ListNamespacesResponse, error) {
	if i.managedNamespace != "" {
		// TODO - add e2e test
		return &infopkg.ListNamespacesResponse{Namespaces: []string{i.managedNamespace}}, nil
	}
	client := auth.GetKubeClient(ctx)
	list, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var namespaces []string
	for _, item := range list.Items {
		// TODO - add e2e test
		namespace := item.Name
		review, err := client.
			AuthorizationV1().
			SelfSubjectAccessReviews().
			Create(ctx, &authv1.SelfSubjectAccessReview{
				Spec: authv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: &authv1.ResourceAttributes{
						// define "namespaces the client is allowed to access" as any they can "list workflows"
						Namespace: namespace,
						Verb:      "list",
						Group:     "argoproj.io",
						Resource:  "workflows",
					},
				},
			}, metav1.CreateOptions{})

		if err != nil {
			return nil, err
		}
		if review.Status.Allowed {
			namespaces = append(namespaces, namespace)
		}
	}
	return &infopkg.ListNamespacesResponse{Namespaces: namespaces}, nil
}

func NewInfoServer(managedNamespace string, links []*wfv1.Link) infopkg.InfoServiceServer {
	return &infoServer{managedNamespace, links}
}
