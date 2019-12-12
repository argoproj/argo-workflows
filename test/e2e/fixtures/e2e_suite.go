package fixtures

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

var kubeConfig = flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "Path to Kubernetes config file")

func init() {
	_ = commands.NewCommand()
}

type E2ESuite struct {
	suite.Suite
	client     v1alpha1.WorkflowInterface
	kubeClient kubernetes.Interface
}

func (suite *E2ESuite) SetupSuite() {
	_, err := os.Stat(*kubeConfig)
	if os.IsNotExist(err) {
		suite.T().Skip("Skipping test: " + err.Error())
		return
	}
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		panic(err)
	}
	suite.kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	suite.client = commands.InitWorkflowClient()
	fmt.Println("deleting workflows")
	timeout := int64(10)
	err = suite.client.DeleteCollection(nil, v1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		panic(err)
	}
}

func (suite *E2ESuite) Given() *Given {
	return &Given{
		t:      suite.T(),
		client: suite.client,
		kubeClient: suite.kubeClient,
	}
}
