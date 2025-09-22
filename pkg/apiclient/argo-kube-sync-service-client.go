package apiclient

import (
	"context"

	"google.golang.org/grpc"

	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
)

type argoKubeSyncServiceClient struct {
	delegate syncpkg.SyncServiceServer
}

var _ syncpkg.SyncServiceClient = &argoKubeSyncServiceClient{}

func (a *argoKubeSyncServiceClient) CreateSyncLimit(ctx context.Context, in *syncpkg.CreateSyncLimitRequest, opts ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	return a.delegate.CreateSyncLimit(ctx, in)
}

func (a *argoKubeSyncServiceClient) DeleteSyncLimit(ctx context.Context, in *syncpkg.DeleteSyncLimitRequest, opts ...grpc.CallOption) (*syncpkg.DeleteSyncLimitResponse, error) {
	return a.delegate.DeleteSyncLimit(ctx, in)
}

func (a *argoKubeSyncServiceClient) GetSyncLimit(ctx context.Context, in *syncpkg.GetSyncLimitRequest, opts ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	return a.delegate.GetSyncLimit(ctx, in)
}

func (a *argoKubeSyncServiceClient) UpdateSyncLimit(ctx context.Context, in *syncpkg.UpdateSyncLimitRequest, opts ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	return a.delegate.UpdateSyncLimit(ctx, in)
}
