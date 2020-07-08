package info

import (
	"context"

	"github.com/argoproj/argo"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
)

type infoServer struct {
	managedNamespace string
	links            []*wfv1.Link
}

func (i *infoServer) WhoAmI(ctx context.Context, _ *infopkg.WhoAmIRequest) (*infopkg.WhoAmIResponse, error) {
	claims := auth.GetClaims(ctx)
	if claims != nil {
		return &infopkg.WhoAmIResponse{Subject: claims.Subject}, nil
	}
	return &infopkg.WhoAmIResponse{}, nil
}

func (i *infoServer) GetInfo(context.Context, *infopkg.GetInfoRequest) (*infopkg.InfoResponse, error) {
	return &infopkg.InfoResponse{ManagedNamespace: i.managedNamespace, Links: i.links}, nil
}

func (i *infoServer) GetVersion(context.Context, *infopkg.GetVersionRequest) (*wfv1.Version, error) {
	version := argo.GetVersion()
	return &version, nil
}

func NewInfoServer(managedNamespace string, links []*wfv1.Link) infopkg.InfoServiceServer {
	return &infoServer{managedNamespace, links}
}
