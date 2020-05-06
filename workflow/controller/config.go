package controller

import (
	"context"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/hydrator"
)

func (wfc *WorkflowController) updateConfig(config config.Config) error {
	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	log.Info("Configuration:\n" + string(bytes))
	if wfc.cliExecutorImage == "" && config.ExecutorImage == "" {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap does not have executorImage")
	}
	wfc.Config = config
	if wfc.session != nil {
		err := wfc.session.Close()
		if err != nil {
			return err
		}
	}
	wfc.session = nil
	wfc.offloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo
	wfc.wfArchive = sqldb.NullWorkflowArchive
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
			wfc.wfArchive = sqldb.NewWorkflowArchive(session, persistence.GetClusterName(), wfc.managedNamespace, instanceIDService)
			log.Info("Workflow archiving is enabled")
		} else {
			log.Info("Workflow archiving is disabled")
		}
	} else {
		log.Info("Persistence configuration disabled")
	}
	wfc.hydrator = hydrator.New(wfc.offloadNodeStatusRepo)
	wfc.throttler.SetParallelism(config.Parallelism)
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

func ReadConfigMapValue(clientset kubernetes.Interface, namespace string, name string, key string) (string, error) {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	value, ok := cm.Data[key]
	if !ok {
		return "", errors.InternalErrorf("Key %s was not found in the %s configMap.", key, name)
	}
	return value, nil
}

func getArtifactRepositoryRef(wfc *WorkflowController, configMapName string, key string, namespace string) (*config.ArtifactRepository, error) {
	// Getting the ConfigMap from the workflow's namespace
	configStr, err := ReadConfigMapValue(wfc.kubeclientset, namespace, configMapName, key)
	if err != nil {
		// Falling back to getting the ConfigMap from the controller's namespace
		configStr, err = ReadConfigMapValue(wfc.kubeclientset, wfc.namespace, configMapName, key)
		if err != nil {
			return nil, err
		}
	}
	var config config.ArtifactRepository
	err = yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &config, nil
}
