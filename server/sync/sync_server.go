package sync

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
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

// NewSyncServer constructs a syncServer that routes synchronization limit operations to provider implementations.
// It always registers a CONFIGMAP provider and, if syncConfig is non-nil with EnableAPI true and a DB session proxy
// can be created via sqldb.NewSessionProxy, also registers a DATABASE provider backed by the provided DBConfig; if
// session proxy creation fails the DATABASE provider is omitted.
func NewSyncServer(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, syncConfig *config.SyncConfig) *syncServer {
	server := &syncServer{
		providers: make(map[syncpkg.SyncConfigType]SyncConfigProvider),
	}

	server.providers[syncpkg.SyncConfigType_CONFIGMAP] = &configMapSyncProvider{}

	if syncConfig != nil && syncConfig.EnableAPI {
		sessionProxy, err := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
			KubectlConfig: kubectlConfig,
			Namespace:     namespace,
			DBConfig:      syncConfig.DBConfig,
		})
		if err == nil {
			server.providers[syncpkg.SyncConfigType_DATABASE] = &dbSyncProvider{db: syncdb.NewSyncQueries(sessionProxy, syncdb.DBConfigFromConfig(syncConfig))}
		}
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