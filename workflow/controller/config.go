package controller

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/time/rate"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3"
	persist "github.com/argoproj/argo-workflows/v3/persist/sqldb"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
	"github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

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
		if wfc.session == nil {
			session, err := sqldb.CreateDBSession(ctx, wfc.kubeclientset, wfc.namespace, persistence.DBConfig)
			if err != nil {
				return err
			}
			logger.Info(ctx, "Persistence Session created successfully")
			wfc.session = session
		}
		sqldb.ConfigureDBSession(wfc.session, persistence.ConnectionPool)
		if os.Getenv("ALWAYS_OFFLOAD_NODE_STATUS") == "true" && !persistence.NodeStatusOffload {
			return errors.New("persistence.NodeStatusOffload must be defined when ALWAYS_OFFLOAD_NODE_STATUS is true")
		}
		if persistence.NodeStatusOffload {
			wfc.offloadNodeStatusRepo, err = persist.NewOffloadNodeStatusRepo(ctx, logger, wfc.session, persistence.GetClusterName(), tableName)
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
			wfc.wfArchive = persist.NewWorkflowArchive(wfc.session, persistence.GetClusterName(), wfc.managedNamespace, instanceIDService)
			logger.Info(ctx, "Workflow archiving is enabled")
		} else {
			logger.Info(ctx, "Workflow archiving is disabled")
		}
	} else {
		logger.Info(ctx, "Persistence configuration disabled")
	}

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

	return persist.Migrate(ctx, wfc.session, persistence.GetClusterName(), tableName)
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
	} else {
		return wfc.Config.GetExecutor().ImagePullPolicy
	}
}
