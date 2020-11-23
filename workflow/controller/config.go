package controller

import (
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist"
	"github.com/argoproj/argo/persist/factory"
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
	wfc.offloadNodeStatusRepo = persist.ExplosiveOffloadNodeStatusRepo
	wfc.wfArchive = persist.NullWorkflowArchive
	wfc.archiveLabelSelector = labels.Everything()
	p, err := factory.New(wfc.kubeclientset, instanceid.NewService(wfc.Config.InstanceID), wfc.namespace, config.Persistence, true)
	if err != nil {
		return err
	}
	wfc.session = p
	wfc.offloadNodeStatusRepo = p.OffloadNodeStatusRepo
	wfc.wfArchive = p.WorkflowArchive
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
