package executor

import (
	"os"
	"path"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	artifact "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/artifacts/git"
	"github.com/argoproj/argo/workflow/artifacts/http"
	"github.com/argoproj/argo/workflow/artifacts/s3"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Executor implements the mechanisms within a single Kubernetes pod
type WorkflowExecutor struct {
	Template  wfv1.Template
	ClientSet *kubernetes.Clientset
}

// Use Kubernetes client to retrieve the Kubernetes secrets
func (we *WorkflowExecutor) getSecrets(namespace string, name string, key string) (string, error) {
	secrets, err := we.ClientSet.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		return "", errors.InternalWrapError(err)
	}

	val, ok := secrets.Data[key]
	if !ok {
		return "", errors.InternalErrorf("Key %s does not exists for secret %s", key, name)
	}
	return string(val), nil
}

func (we *WorkflowExecutor) LoadArtifacts() error {
	log.Infof("Start loading input artifacts...")

	for _, art := range we.Template.Inputs.Artifacts {
		log.Infof("Downloading artifact: %s", art.Name)
		artDriver, err := we.InitDriver(art)
		if err != nil {
			return err
		}
		// File path to save the artifact
		// TODO: need to choose a different path if the path overlaps with a volume mount
		artPath := path.Join(common.ExecutorArtifactBaseDir, art.Name)

		err = artDriver.Load(&art, artPath)
		if err != nil {
			return err
		}
		log.Infof("Successfully download file: %s", artPath)
		if art.Mode != nil {
			err = os.Chmod(artPath, os.FileMode(*art.Mode))
			if err != nil {
				return errors.InternalWrapError(err)
			}
		}
	}

	return nil
}

func (we *WorkflowExecutor) InitDriver(art wfv1.Artifact) (artifact.ArtifactDriver, error) {
	if art.S3 != nil {
		// Getting Kubernetes namespace from the environment variables
		namespace := os.Getenv(common.EnvVarNamespace)
		accessKey, err := we.getSecrets(namespace, art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
		if err != nil {
			return nil, err
		}
		secretKey, err := we.getSecrets(namespace, art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
		if err != nil {
			return nil, err
		}
		driver := s3.S3ArtifactDriver{
			AccessKey: accessKey,
			SecretKey: secretKey,
		}
		return &driver, nil
	}
	if art.HTTP != nil {
		return &http.HTTPArtifactDriver{URL: art.HTTP.URL}, nil
	}
	if art.Git != nil {
		return &git.GitArtifactDriver{Repo: art.Git.Repo, Revision: art.Git.Revision}, nil
	}
	return nil, errors.Errorf(errors.CodeBadRequest, "Unsupported artifact driver for %s", art.Name)
}
