package e2e

import (
	"fmt"
	"testing"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/argoproj/argo/workflow/common"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func checkIfInstalled() bool {
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
	if !checkIfInstalled() {
		commands.Install(nil, nil)
	}
}
