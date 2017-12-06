package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/argoproj/argo/workflow/common"
	"github.com/stretchr/testify/assert"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func checkIfInstalled(namespace string) bool {
	clientSet := getKubernetesClient()

	// TODO(shri): Create a new namespace and simply install in that.
	// Verify that Argo doesn't exist in the Kube-system namespace
	_, err := clientSet.AppsV1beta2().Deployments(common.DefaultControllerNamespace).Get(
		common.DefaultControllerDeploymentName, metav1.GetOptions{})
	if err == nil {
		fmt.Println("Argo already installed...")
		return true
	}

	if err != nil {
		if !apierr.IsNotFound(err) {
			panic(err)
		}
	}

	return false
}

func TestInstall(t *testing.T) {
	namespace := "default"
	if !checkIfInstalled(namespace) {
		args := commands.InstallFlags{
			ControllerName: common.DefaultControllerDeploymentName,
			UIName:         common.DefaultUiDeploymentName,
			Namespace:      namespace,
			ConfigMap:      common.DefaultConfigMapName(common.DefaultControllerDeploymentName),
			//TODO(shri): Use better defaults that don't need Makefiles
			ControllerImage: "argoproj/workflow-controller:v2.0.0-alpha2",
			UIImage:         "argoproj/argoui:v2.0.0-alpha2",
			ExecutorImage:   "argoproj/argoexec:v2.0.0-alpha2",
			ServiceAccount:  "",
		}

		commands.Install(nil, args)
		// Wait a little for the installation to complete.
		time.Sleep(10 * time.Second)
		assert.Equal(t, true, checkIfInstalled(namespace))
	}
}
