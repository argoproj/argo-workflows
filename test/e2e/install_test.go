package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/argoproj/argo/install"
	"github.com/argoproj/argo/workflow/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstallSuite struct {
	suite.Suite
	testNamespace string
}

func (suite *InstallSuite) SetupSuite() {
	if *kubeConfig == "" {
		suite.T().Skip("Skipping test. Kubeconfig not provided")
	}
}

// Make sure that a new namespace is created before each test
func (suite *InstallSuite) SetupTest() {
	suite.testNamespace = createNamespaceForTest()
}

func (suite *InstallSuite) TearDownTest() {
	if err := deleteTestNamespace(suite.testNamespace); err != nil {
		panic(err)
	}
}

func checkIfInstalled(namespace string) bool {
	_, clientSet := getKubernetesClient()

	// Verify that Argo doesn't exist in the Kube-system namespace
	_, err := clientSet.AppsV1beta2().Deployments(namespace).Get(
		common.DefaultControllerDeploymentName, metav1.GetOptions{})
	if err == nil {
		fmt.Printf("Argo already installed in namespace %s...\n", namespace)
		return true
	}

	if err != nil {
		if !apierr.IsNotFound(err) {
			panic(err)
		}
	}

	return false
}

func (suite *InstallSuite) TestInstall() {
	t := suite.T()
	if !checkIfInstalled(suite.testNamespace) {
		// Verify --dry-run doesn't install
		args := newInstallArgs(suite.testNamespace)
		args.DryRun = true
		config, _ := getKubernetesClient()
		installer, err := install.NewInstaller(config, args)
		if err != nil {
			panic(err)
		}
		installer.Install()
		assert.Equal(t, false, checkIfInstalled(suite.testNamespace))

		installArgoInNamespace(suite.testNamespace)
		// Wait a little for the installation to complete.
		time.Sleep(10 * time.Second)
		assert.Equal(t, true, checkIfInstalled(suite.testNamespace))
	}
}

func TestArgoInstall(t *testing.T) {
	suite.Run(t, new(InstallSuite))
}
