package controller

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

func (wfc *WorkflowController) updateConfig() error {
	bytes, err := yaml.Marshal(wfc.Config)
	if err != nil {
		return err
	}
	log.Info("Configuration:\n" + string(bytes))
	wfc.session = nil
	wfc.artifactRepositories = artifactrepositories.New(wfc.kubeclientset, wfc.namespace, &wfc.Config.ArtifactRepository)
	wfc.offloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo
	wfc.wfArchive = sqldb.NullWorkflowArchive
	wfc.archiveLabelSelector = labels.Everything()
	persistence := wfc.Config.Persistence
	if persistence != nil {
		log.Info("Persistence configuration enabled")
		session, tableName, err := sqldb.CreateDBSession(wfc.kubeclientset, wfc.namespace, persistence)
		if err != nil {
			return err
		}
		log.Info("Persistence Session created successfully")
		if !persistence.SkipMigration {
			err = sqldb.NewMigrate(session, persistence.GetClusterName(), tableName).Exec(context.Background())
			if err != nil {
				return err
			}
		} else {
			log.Info("DB migration is disabled")
		}

		wfc.session = session
		if persistence.NodeStatusOffload {
			wfc.offloadNodeStatusRepo, err = sqldb.NewOffloadNodeStatusRepo(session, persistence.GetClusterName(), tableName)
			if err != nil {
				return err
			}
			log.Info("Node status offloading is enabled")
		} else {
			log.Info("Node status offloading is disabled")
		}
		if persistence.Archive {
			instanceIDService := instanceid.NewService(wfc.Config.InstanceID)

			wfc.archiveLabelSelector, err = persistence.GetArchiveLabelSelector()
			if err != nil {
				return err
			}
			wfc.wfArchive = sqldb.NewWorkflowArchive(session, persistence.GetClusterName(), wfc.managedNamespace, instanceIDService)
			log.Info("Workflow archiving is enabled")
		} else {
			log.Info("Workflow archiving is disabled")
		}
	} else {
		log.Info("Persistence configuration disabled")
	}
	wfc.hydrator = hydrator.New(wfc.offloadNodeStatusRepo)
	wfc.updateEstimatorFactory()
	wfc.rateLimiter = wfc.newRateLimiter()

	log.WithField("executorImage", wfc.executorImage()).
		WithField("executorImagePullPolicy", wfc.executorImagePullPolicy()).
		WithField("managedNamespace", wfc.GetManagedNamespace()).
		Info()
	return nil
}

func (wfc *WorkflowController) newRateLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(wfc.Config.GetResourceRateLimit().Limit), wfc.Config.GetResourceRateLimit().Burst)
}

// executorImage returns the image to use for the workflow executor
func (wfc *WorkflowController) executorImage() string {
	if wfc.cliExecutorImage != "" {
		return wfc.cliExecutorImage
	}
	if v := wfc.Config.GetExecutor().Image; v != "" {
		return v
	}
	return fmt.Sprintf("quay.io/argoproj/argoexec:" + argo.ImageTag())
}

// executorImagePullPolicy returns the imagePullPolicy to use for the workflow executor
func (wfc *WorkflowController) executorImagePullPolicy() apiv1.PullPolicy {
	if wfc.cliExecutorImagePullPolicy != "" {
		return apiv1.PullPolicy(wfc.cliExecutorImagePullPolicy)
	} else {
		return wfc.Config.GetExecutor().ImagePullPolicy
	}
}
