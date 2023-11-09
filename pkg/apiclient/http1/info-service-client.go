package http1

import (
	"context"

	"google.golang.org/grpc"

	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type InfoServiceClient = Facade

func (h InfoServiceClient) GetInfo(_ context.Context, in *infopkg.GetInfoRequest, _ ...grpc.CallOption) (*infopkg.InfoResponse, error) {
	out := &infopkg.InfoResponse{}
	return out, h.Get(in, out, "/api/v1/info")
}

func (h InfoServiceClient) GetVersion(_ context.Context, in *infopkg.GetVersionRequest, _ ...grpc.CallOption) (*wfv1.Version, error) {
	out := &wfv1.Version{}
	return out, h.Get(in, out, "/api/v1/version")
}

func (h InfoServiceClient) GetUserInfo(_ context.Context, in *infopkg.GetUserInfoRequest, _ ...grpc.CallOption) (*infopkg.GetUserInfoResponse, error) {
	out := &infopkg.GetUserInfoResponse{}
	return out, h.Get(in, out, "/api/v1/userinfo")
}

func (h InfoServiceClient) CollectEvent(_ context.Context, in *infopkg.CollectEventRequest, _ ...grpc.CallOption) (*infopkg.CollectEventResponse, error) {
	out := &infopkg.CollectEventResponse{}
	return out, h.Post(in, out, "/api/v1/tracking/event")
}
