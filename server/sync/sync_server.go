package sync

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	syncdb "github.com/argoproj/argo-workflows/v3/util/sync/db"
)

type ConfigProvider interface {
	createSyncLimit(ctx context.Context, req *syncpkg.CreateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error)
	getSyncLimit(ctx context.Context, req *syncpkg.GetSyncLimitRequest) (*syncpkg.SyncLimitResponse, error)
	updateSyncLimit(ctx context.Context, req *syncpkg.UpdateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error)
	deleteSyncLimit(ctx context.Context, req *syncpkg.DeleteSyncLimitRequest) (*syncpkg.DeleteSyncLimitResponse, error)
}

type syncServer struct {
	providers map[syncpkg.SyncConfigType]ConfigProvider
}

func NewSyncServer(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, syncConfig *config.SyncConfig) syncpkg.SyncServiceServer {
	server := &syncServer{
		providers: make(map[syncpkg.SyncConfigType]ConfigProvider),
	}

	server.providers[syncpkg.SyncConfigType_CONFIGMAP] = &configMapSyncProvider{}

	if syncConfig != nil && syncConfig.EnableAPI {
		session, _ := syncdb.SessionFromConfig(ctx, kubectlConfig, namespace, syncConfig)
		server.providers[syncpkg.SyncConfigType_DATABASE] = &dbSyncProvider{db: syncdb.NewSyncQueries(session, syncdb.ConfigFromConfig(syncConfig))}
	}

	return server
}

func (s *syncServer) CreateSyncLimit(ctx context.Context, req *syncpkg.CreateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	if req.Limit <= 0 {
		return nil, sutils.ToStatusError(fmt.Errorf("limit must be greater than zero"), codes.InvalidArgument)
	}

	provider, ok := s.providers[req.Type]
	if !ok {
		return nil, sutils.ToStatusError(fmt.Errorf("unsupported sync config type: %s", req.Type), codes.InvalidArgument)
	}
	return provider.createSyncLimit(ctx, req)
}

func (s *syncServer) GetSyncLimit(ctx context.Context, req *syncpkg.GetSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	provider, ok := s.providers[req.Type]
	if !ok {
		return nil, sutils.ToStatusError(fmt.Errorf("unsupported sync config type: %s", req.Type), codes.InvalidArgument)
	}
	return provider.getSyncLimit(ctx, req)
}

func (s *syncServer) UpdateSyncLimit(ctx context.Context, req *syncpkg.UpdateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	if req.Limit <= 0 {
		return nil, sutils.ToStatusError(fmt.Errorf("limit must be greater than zero"), codes.InvalidArgument)
	}

	provider, ok := s.providers[req.Type]
	if !ok {
		return nil, sutils.ToStatusError(fmt.Errorf("unsupported sync config type: %s", req.Type), codes.InvalidArgument)
	}
	return provider.updateSyncLimit(ctx, req)
}

func (s *syncServer) DeleteSyncLimit(ctx context.Context, req *syncpkg.DeleteSyncLimitRequest) (*syncpkg.DeleteSyncLimitResponse, error) {
	provider, ok := s.providers[req.Type]
	if !ok {
		return nil, sutils.ToStatusError(fmt.Errorf("unsupported sync config type: %s", req.Type), codes.InvalidArgument)
	}
	return provider.deleteSyncLimit(ctx, req)
}
