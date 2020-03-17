package info

import (
	"context"

	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type infoServer struct {
	managedNamespace string
	links            []*wfv1.Link
}

func (i *infoServer) GetInfo(context.Context, *infopkg.GetInfoRequest) (*infopkg.InfoResponse, error) {
	return &infopkg.InfoResponse{ManagedNamespace: i.managedNamespace, Links: i.links}, nil
}

func NewInfoServer(managedNamespace string, links []*wfv1.Link) infopkg.InfoServiceServer {
	return &infoServer{managedNamespace, links}
}
