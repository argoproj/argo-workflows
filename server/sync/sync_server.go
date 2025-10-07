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

type SyncConfigProvider interface {
	createSyncLimit(ctx context.Context, req *syncpkg.CreateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error)
	getSyncLimit(ctx context.Context, req *syncpkg.GetSyncLimitRequest) (*syncpkg.SyncLimitResponse, error)
	updateSyncLimit(ctx context.Context, req *syncpkg.UpdateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error)
	deleteSyncLimit(ctx context.Context, req *syncpkg.DeleteSyncLimitRequest) (*syncpkg.DeleteSyncLimitResponse, error)
}

type syncServer struct {
	providers map[syncpkg.SyncConfigType]SyncConfigProvider
}

func NewSyncServer(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, dbConfig *config.SyncConfig) *syncServer {
	server := &syncServer{
		providers: make(map[syncpkg.SyncConfigType]SyncConfigProvider),
	}

	server.providers[syncpkg.SyncConfigType_CONFIGMAP] = &configMapSyncProvider{}

	if dbConfig != nil && (dbConfig.MySQL != nil || dbConfig.PostgreSQL != nil) {
		session := syncdb.DBSessionFromConfig(ctx, kubectlConfig, namespace, dbConfig)
		server.providers[syncpkg.SyncConfigType_DATABASE] = &dbSyncProvider{db: syncdb.NewSyncQueries(session, syncdb.DBConfigFromConfig(dbConfig))}
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
