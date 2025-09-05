package apiclient

import (
	"context"

	"google.golang.org/grpc"

	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	grpcutil "github.com/argoproj/argo-workflows/v3/util/grpc"
)

type errorTranslatingArgoKubeSyncServiceClient struct {
	delegate syncpkg.SyncServiceClient
}

var _ syncpkg.SyncServiceClient = &errorTranslatingArgoKubeSyncServiceClient{}

func (e *errorTranslatingArgoKubeSyncServiceClient) CreateSyncLimit(ctx context.Context, in *syncpkg.CreateSyncLimitRequest, opts ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	syncLimit, err := e.delegate.CreateSyncLimit(ctx, in, opts...)
	return syncLimit, grpcutil.TranslateError(err)
}

func (e *errorTranslatingArgoKubeSyncServiceClient) DeleteSyncLimit(ctx context.Context, in *syncpkg.DeleteSyncLimitRequest, opts ...grpc.CallOption) (*syncpkg.DeleteSyncLimitResponse, error) {
	deleteResp, err := e.delegate.DeleteSyncLimit(ctx, in, opts...)
	return deleteResp, grpcutil.TranslateError(err)
}

func (e *errorTranslatingArgoKubeSyncServiceClient) GetSyncLimit(ctx context.Context, in *syncpkg.GetSyncLimitRequest, opts ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	syncLimit, err := e.delegate.GetSyncLimit(ctx, in, opts...)
	return syncLimit, grpcutil.TranslateError(err)
}

func (e *errorTranslatingArgoKubeSyncServiceClient) UpdateSyncLimit(ctx context.Context, in *syncpkg.UpdateSyncLimitRequest, opts ...grpc.CallOption) (*syncpkg.SyncLimitResponse, error) {
	syncLimit, err := e.delegate.UpdateSyncLimit(ctx, in, opts...)
	return syncLimit, grpcutil.TranslateError(err)
}
