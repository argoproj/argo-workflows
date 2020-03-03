package info

import (
	"context"

	infopkg "github.com/argoproj/argo/v2/pkg/apiclient/info"
)

type infoServer struct {
	managedNamespace string
}

func (i *infoServer) GetInfo(context.Context, *infopkg.GetInfoRequest) (*infopkg.InfoResponse, error) {
	return &infopkg.InfoResponse{ManagedNamespace: i.managedNamespace}, nil
}

func NewInfoServer(managedNamespace string) infopkg.InfoServiceServer {
	return &infoServer{managedNamespace}
}
