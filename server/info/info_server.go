package info

import (
	"context"

	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
)

type infoServer struct {
	managedNamespace string
	links            []*wfv1.Link
}

func (i *infoServer) GetInfo(ctx context.Context, _ *infopkg.GetInfoRequest) (*infopkg.InfoResponse, error) {
	user := auth.GetUser(ctx)
	return &infopkg.InfoResponse{ManagedNamespace: i.managedNamespace, Links: i.links, User: &user}, nil
}

func NewInfoServer(managedNamespace string, links []*wfv1.Link) infopkg.InfoServiceServer {
	return &infoServer{managedNamespace, links}
}
