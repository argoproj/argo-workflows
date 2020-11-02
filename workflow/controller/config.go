package controller

import (
	"context"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/hydrator"
)

func (wfc *WorkflowController) updateConfig(v interface{}) error {
	config := v.(*config.Config)
	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	log.Info("Configuration:\n" + string(bytes))
	if wfc.cliExecutorImage == "" && config.ExecutorImage == "" {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap does not have executorImage")
	}
	wfc.Config = *config
	if wfc.session != nil {
		err := wfc.session.Close()
		if err != nil {
			return err
		}
	}
	wfc.session = nil
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
		err = sqldb.NewMigrate(session, persistence.GetClusterName(), tableName).Exec(context.Background())
		if err != nil {
			return err
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
	return nil
}

// executorImage returns the image to use for the workflow executor
func (wfc *WorkflowController) executorImage() string {
	if wfc.cliExecutorImage != "" {
		return wfc.cliExecutorImage
	}
	return wfc.Config.ExecutorImage
}

// executorImagePullPolicy returns the imagePullPolicy to use for the workflow executor
func (wfc *WorkflowController) executorImagePullPolicy() apiv1.PullPolicy {
	if wfc.cliExecutorImagePullPolicy != "" {
		return apiv1.PullPolicy(wfc.cliExecutorImagePullPolicy)
	} else if wfc.Config.Executor != nil && wfc.Config.Executor.ImagePullPolicy != "" {
		return wfc.Config.Executor.ImagePullPolicy
	} else {
		return apiv1.PullPolicy(wfc.Config.ExecutorImagePullPolicy)
	}
}
