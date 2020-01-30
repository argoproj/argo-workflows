package info

import "context"

type infoServer struct {
	managedNamespace string
}

func (i *infoServer) GetInfo(context.Context, *GetInfoRequest) (*InfoResponse, error) {
	return &InfoResponse{ManagedNamespace: i.managedNamespace}, nil
}

func NewInfoServer(managedNamespace string) InfoServiceServer {
	return &infoServer{managedNamespace}
}
