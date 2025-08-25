package sync

import (
	"context"
	"fmt"

	"github.com/upper/db/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	syncdb "github.com/argoproj/argo-workflows/v3/util/sync/db"
)

type dbSyncProvider struct {
	db syncdb.SyncQueries
}

var _ SyncConfigProvider = &dbSyncProvider{}

func (s *dbSyncProvider) createSyncLimit(ctx context.Context, req *syncpkg.CreateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	allowed, err := auth.CanI(ctx, "create", workflow.WorkflowPlural, req.Namespace, "")
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Permission denied, you are not allowed to create database sync limit in namespace \"%s\".", req.Namespace))
	}

	name := fmt.Sprintf("%s/%s", req.Namespace, req.Key)
	_, err = s.db.GetSemaphoreLimit(ctx, name)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("Database sync limit already exists in namespace \"%s\".", req.Namespace))
	} else if err != db.ErrNoMoreRows {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	err = s.db.CreateSemaphoreLimit(ctx, name, int(req.SizeLimit))
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &syncpkg.SyncLimitResponse{Key: req.Key, Namespace: req.Namespace, SizeLimit: req.SizeLimit}, nil
}

func (s *dbSyncProvider) getSyncLimit(ctx context.Context, req *syncpkg.GetSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	allowed, err := auth.CanI(ctx, "get", workflow.WorkflowPlural, req.Namespace, "")
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Permission denied, you are not allowed to get database sync limit in namespace \"%s\".", req.Namespace))
	}

	name := fmt.Sprintf("%s/%s", req.Namespace, req.Key)
	limit, err := s.db.GetSemaphoreLimit(ctx, name)
	if err != nil {
		if err == db.ErrNoMoreRows {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("Database sync limit not found in namespace \"%s\".", req.Namespace))
		}
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &syncpkg.SyncLimitResponse{Key: req.Key, Namespace: req.Namespace, SizeLimit: int32(limit.SizeLimit)}, nil
}

func (s *dbSyncProvider) updateSyncLimit(ctx context.Context, req *syncpkg.UpdateSyncLimitRequest) (*syncpkg.SyncLimitResponse, error) {
	allowed, err := auth.CanI(ctx, "update", workflow.WorkflowPlural, req.Namespace, "")
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Permission denied, you are not allowed to update database sync limit in namespace \"%s\".", req.Namespace))
	}

	name := fmt.Sprintf("%s/%s", req.Namespace, req.Key)
	err = s.db.UpdateSemaphoreLimit(ctx, name, int(req.SizeLimit))
	if err != nil {
		if err == db.ErrNoMoreRows {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("Database sync limit not found in namespace \"%s\".", req.Namespace))
		}
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &syncpkg.SyncLimitResponse{Key: req.Key, Namespace: req.Namespace, SizeLimit: req.SizeLimit}, nil
}

func (s *dbSyncProvider) deleteSyncLimit(ctx context.Context, req *syncpkg.DeleteSyncLimitRequest) (*syncpkg.DeleteSyncLimitResponse, error) {
	allowed, err := auth.CanI(ctx, "delete", workflow.WorkflowPlural, req.Namespace, "")
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Permission denied, you are not allowed to delete database sync limit in namespace \"%s\".", req.Namespace))
	}

	// we don't care if semaphore is in use
	// wc should be able to recover
	name := fmt.Sprintf("%s/%s", req.Namespace, req.Key)
	err = s.db.DeleteSemaphoreLimit(ctx, name)
	if err != nil {
		if err == db.ErrNoMoreRows {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("Database sync limit not found in namespace \"%s\".", req.Namespace))
		}
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &syncpkg.DeleteSyncLimitResponse{}, nil
}
