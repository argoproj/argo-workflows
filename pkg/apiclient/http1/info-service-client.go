package http1

import (
	"context"

	"google.golang.org/grpc"

	infopkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/info"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type InfoServiceClient = Facade

func (h InfoServiceClient) GetInfo(ctx context.Context, in *infopkg.GetInfoRequest, _ ...grpc.CallOption) (*infopkg.InfoResponse, error) {
	out := &infopkg.InfoResponse{}
	return out, h.Get(ctx, in, out, "/api/v1/info")
}

func (h InfoServiceClient) GetVersion(ctx context.Context, in *infopkg.GetVersionRequest, _ ...grpc.CallOption) (*wfv1.Version, error) {
	out := &wfv1.Version{}
	return out, h.Get(ctx, in, out, "/api/v1/version")
}

func (h InfoServiceClient) GetUserInfo(ctx context.Context, in *infopkg.GetUserInfoRequest, _ ...grpc.CallOption) (*infopkg.GetUserInfoResponse, error) {
	out := &infopkg.GetUserInfoResponse{}
	return out, h.Get(ctx, in, out, "/api/v1/userinfo")
}

func (h InfoServiceClient) CollectEvent(ctx context.Context, in *infopkg.CollectEventRequest, _ ...grpc.CallOption) (*infopkg.CollectEventResponse, error) {
	out := &infopkg.CollectEventResponse{}
	return out, h.Post(ctx, in, out, "/api/v1/tracking/event")
}
