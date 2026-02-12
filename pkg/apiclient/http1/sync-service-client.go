package http1

import (
	"context"

	"google.golang.org/grpc"

	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
)

type SyncServiceClient = Facade

func (h SyncServiceClient) GetSyncLimit(ctx context.Context, in *syncpkg.GetSyncLimitRequest, _ ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	out := &syncpkg.SyncLimitResponse{}
	return out, h.Get(ctx, in, out, "/api/v1/sync/{namespace}/{key}")
}

func (h SyncServiceClient) CreateSyncLimit(ctx context.Context, in *syncpkg.CreateSyncLimitRequest, _ ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	out := &syncpkg.SyncLimitResponse{}
	return out, h.Post(ctx, in, out, "/api/v1/sync/{namespace}")
}

func (h SyncServiceClient) DeleteSyncLimit(ctx context.Context, in *syncpkg.DeleteSyncLimitRequest, _ ...grpc.CallOption) (*syncpkg.DeleteSyncLimitResponse, error) {
	out := &syncpkg.DeleteSyncLimitResponse{}
	return out, h.Delete(ctx, in, out, "/api/v1/sync/{namespace}/{key}")
}

func (h SyncServiceClient) UpdateSyncLimit(ctx context.Context, in *syncpkg.UpdateSyncLimitRequest, _ ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	out := &syncpkg.SyncLimitResponse{}
	return out, h.Put(ctx, in, out, "/api/v1/sync/{namespace}/{key}")
}
