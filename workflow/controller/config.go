package controller

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4"
	persist "github.com/argoproj/argo-workflows/v4/persist/sqldb"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	"github.com/argoproj/argo-workflows/v4/workflow/artifactrepositories"
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
)

var (
	memoSessionProxyFromConfig = memodb.SessionProxyFromConfig
	memoizationMigrate         = memodb.Migrate
)

// initializeMemoizationBackendLocked creates and migrates the memoization SQL backend.
// The caller must hold wfc.memoLock.
func (wfc *WorkflowController) initializeMemoizationBackendLocked(ctx context.Context, logger logging.Logger) {
	if wfc.memoConfig == nil {
		return
	}
	if wfc.memoSessionProxy == nil {
		sp := memoSessionProxyFromConfig(ctx, wfc.kubeclientset, wfc.namespace, wfc.memoConfig)
		if sp == nil {
			logger.Error(ctx, "Failed to connect to memoization database; SQL caching unavailable. Workflows using memoization will skip caching until the database is reachable.")
			wfc.memoSessionProxy = nil
			wfc.memoMigrated = false
			wfc.cacheFactory.ClearSessionProxy(true)
			return
		}
		wfc.memoSessionProxy = sp
		wfc.memoMigrated = false
	}
	if wfc.memoSessionProxy != nil && !wfc.memoMigrated {
		cfg := memodb.ConfigFromConfig(wfc.memoConfig)
		if err := memoizationMigrate(ctx, wfc.memoSessionProxy, cfg); err != nil {
			logger.WithError(err).Error(ctx, "Memoization database migration failed; SQL caching unavailable. Workflows using memoization will skip caching until the database is healthy.")
			wfc.memoSessionProxy.Close()
			wfc.memoSessionProxy = nil
			wfc.memoMigrated = false
			wfc.cacheFactory.ClearSessionProxy(true)
			return
		}
		wfc.memoMigrated = true
		wfc.cacheFactory.SetSessionProxy(wfc.memoSessionProxy, cfg.TableName)
	}
}

// ensureMemoizationBackend makes sure the configured memoization SQL backend is ready for use.
// It returns true when memoization can proceed against the configured backend.
func (wfc *WorkflowController) ensureMemoizationBackend(ctx context.Context) bool {
	logger := logging.RequireLoggerFromContext(ctx)
	wfc.memoLock.Lock()
	defer wfc.memoLock.Unlock()
	if wfc.memoConfig == nil {
		return true
	}
	if wfc.memoSessionProxy != nil && wfc.memoMigrated {
		return true
	}
	wfc.initializeMemoizationBackendLocked(ctx, logger)
	return wfc.memoSessionProxy != nil && wfc.memoMigrated
}

// getMemoizationCache returns the memoization cache for the given namespace and name.
// When SQL memoization is configured, it first ensures the backend is available.
func (wfc *WorkflowController) getMemoizationCache(ctx context.Context, namespace, name string) controllercache.MemoizationCache {
	wfc.memoLock.RLock()
	memoConfigured := wfc.memoConfig != nil
	wfc.memoLock.RUnlock()
	if memoConfigured && !wfc.ensureMemoizationBackend(ctx) {
		return nil
	}
	return wfc.cacheFactory.GetCache(ctx, controllercache.ConfigMapCache, namespace, name)
}

func (wfc *WorkflowController) updateConfig(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	_, err := yaml.Marshal(wfc.Config)
	if err != nil {
		return err
	}
	logger.Info(ctx, "Configuration updated")
	wfc.artifactRepositories = artifactrepositories.New(wfc.kubeclientset, wfc.namespace, &wfc.Config.ArtifactRepository)
	wfc.offloadNodeStatusRepo = persist.ExplosiveOffloadNodeStatusRepo
	wfc.wfArchive = persist.NullWorkflowArchive
	wfc.archiveLabelSelector = labels.Everything()
	if wfc.throttler != nil {
		wfc.throttler.UpdateParallelism(wfc.Config.Parallelism)
	}

	persistence := wfc.Config.Persistence
	if persistence != nil {
		logger.Info(ctx, "Persistence configuration enabled")
		tableName, err := persist.GetTableName(persistence)
		if err != nil {
			return err
		}
		if wfc.sessionProxy == nil {
			sessionProxy, sessionErr := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
				KubectlConfig: wfc.kubeclientset,
				Namespace:     wfc.namespace,
				DBConfig:      persistence.DBConfig,
			})
			if sessionErr != nil {
				return sessionErr
			}
			logger.Info(ctx, "Persistence Session created successfully")
			wfc.sessionProxy = sessionProxy
		}
		sqldb.ConfigureDBSession(wfc.sessionProxy.Session(), persistence.ConnectionPool)
		if persistence.NodeStatusOffload {
			wfc.offloadNodeStatusRepo, err = persist.NewOffloadNodeStatusRepo(ctx, logger, wfc.sessionProxy, persistence.GetClusterName(), tableName)
			if err != nil {
				return err
			}
			logger.Info(ctx, "Node status offloading is enabled")
		} else {
			logger.Info(ctx, "Node status offloading is disabled")
		}
		if persistence.Archive {
			instanceIDService := instanceid.NewService(wfc.Config.InstanceID)

			wfc.archiveLabelSelector, err = persistence.GetArchiveLabelSelector()
			if err != nil {
				return err
			}
			wfc.wfArchive = persist.NewWorkflowArchive(wfc.sessionProxy, persistence.GetClusterName(), wfc.managedNamespace, instanceIDService)
			logger.Info(ctx, "Workflow archiving is enabled")
		} else {
			logger.Info(ctx, "Workflow archiving is disabled")
		}
	} else {
		logger.Info(ctx, "Persistence configuration disabled")
	}

	memoCfg := wfc.Config.Memoization
	wfc.memoLock.Lock()
	wfc.memoConfig = memoCfg
	if memoCfg != nil {
		logger.Info(ctx, "Memoization database configuration enabled")
		wfc.initializeMemoizationBackendLocked(ctx, logger)
	} else {
		if wfc.memoSessionProxy != nil {
			logger.Info(ctx, "Memoization database configuration removed")
			wfc.memoSessionProxy.Close()
			wfc.memoSessionProxy = nil
			wfc.memoMigrated = false
		}
		wfc.cacheFactory.ClearSessionProxy(false)
		logger.Info(ctx, "Memoization database configuration disabled; using ConfigMap-based caching")
	}
	wfc.memoLock.Unlock()

	wfc.hydrator = hydrator.New(wfc.offloadNodeStatusRepo)
	wfc.updateEstimatorFactory(ctx)
	wfc.rateLimiter = wfc.newRateLimiter()
	wfc.maxStackDepth = wfc.getMaxStackDepth()

	logger.WithField("executorImage", wfc.executorImage()).
		WithField("executorImagePullPolicy", wfc.executorImagePullPolicy()).
		WithField("managedNamespace", wfc.GetManagedNamespace()).
		Info(ctx, "")
	return nil
}

// initDB inits argo DB tables
func (wfc *WorkflowController) initDB(ctx context.Context) error {
	persistence := wfc.Config.Persistence
	if persistence == nil || persistence.SkipMigration {
		logger := logging.RequireLoggerFromContext(ctx)
		logger.Info(ctx, "DB migration is disabled")
		return nil
	}
	tableName, err := persist.GetTableName(persistence)
	if err != nil {
		return err
	}

	return persist.Migrate(ctx, wfc.sessionProxy.Session(), persistence.GetClusterName(), tableName, wfc.sessionProxy.DBType())
}

func (wfc *WorkflowController) newRateLimiter() *rate.Limiter {
	rateLimiter := wfc.Config.GetResourceRateLimit()
	return rate.NewLimiter(rate.Limit(rateLimiter.Limit), rateLimiter.Burst)
}

// executorImage returns the image to use for the workflow executor
func (wfc *WorkflowController) executorImage() string {
	if wfc.cliExecutorImage != "" {
		return wfc.cliExecutorImage
	}
	if v := wfc.Config.GetExecutor().Image; v != "" {
		return v
	}
	return fmt.Sprintf("quay.io/argoproj/argoexec:%s", argo.ImageTag())
}

func (wfc *WorkflowController) executorLogFormat() string {
	return wfc.cliExecutorLogFormat
}

// executorImagePullPolicy returns the imagePullPolicy to use for the workflow executor
func (wfc *WorkflowController) executorImagePullPolicy() apiv1.PullPolicy {
	if wfc.cliExecutorImagePullPolicy != "" {
		return apiv1.PullPolicy(wfc.cliExecutorImagePullPolicy)
	}
	return wfc.Config.GetExecutor().ImagePullPolicy
}
