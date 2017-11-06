package executor

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	artifact "github.com/argoproj/argo/workflow/artifacts"
	"github.com/argoproj/argo/workflow/artifacts/s3"
	apiv1 "k8s.io/api/core/v1"
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

	return string(secrets.Data[key]), nil
}

func (we *WorkflowExecutor) LoadArtifacts() error {
	fmt.Println("Start loading input artifacts...")

	// Todo get the current namespace from environment variable
	namespace := apiv1.NamespaceDefault
	for _, art := range we.Template.Inputs.Artifacts {
		fmt.Printf("Downloading artifact, %s\n", art.Name)
		var artDriver artifact.ArtifactDriver
		if art.S3 != nil {
			accessKey, err := we.getSecrets(namespace, art.S3.AccessKeySecret.Name, art.S3.AccessKeySecret.Key)
			if err != nil {
				return err
			}
			secretKey, err := we.getSecrets(namespace, art.S3.SecretKeySecret.Name, art.S3.SecretKeySecret.Key)
			if err != nil {
				return err
			}
			artDriver = &s3.S3ArtifactDriver{
				AccessKey: accessKey,
				SecretKey: secretKey,
			}
		} else {
			fmt.Printf("Do not support input artifact type other than S3, did not download artifact, %s/n", art.Name)
			// Todo currently only support S3
			//return errors.Errorf(errors.CodeInternal, "Do not support input artifact type other than S3 for artifact, %s", artName)
			return nil
		}

		err := artDriver.Load(&art)
		if err != nil {
			return err
		}
	}

	return nil
}
