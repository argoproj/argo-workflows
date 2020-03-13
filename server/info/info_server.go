package info

import (
	"context"

	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type infoServer struct {
	managedNamespace string
	loggingFacility  wfv1.LoggingFacility
}

func (i *infoServer) GetInfo(context.Context, *infopkg.GetInfoRequest) (*infopkg.InfoResponse, error) {
	return &infopkg.InfoResponse{
		ManagedNamespace: i.managedNamespace,
		LoggingFacility:  &i.loggingFacility,
	}, nil
}

func NewInfoServer(managedNamespace string, loggingFacility wfv1.LoggingFacility) infopkg.InfoServiceServer {
	return &infoServer{managedNamespace, loggingFacility}
}
