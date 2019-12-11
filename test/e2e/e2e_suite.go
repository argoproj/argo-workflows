package e2e

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	kube      *kubernetes.Clientset
	wf        v1alpha1.WorkflowInterface
	namespace string
}

func (suite *E2ESuite) SetupSuite() {
	_, err := os.Stat(*kubeConfig)
	if os.IsNotExist(err) {
		suite.T().Skip("Skipping test: " + err.Error())
		return
	}
	_, suite.kube = getKubernetesClient()
	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "argo-e2e-test-"}}
	cns, err := suite.kube.CoreV1().Namespaces().Create(ns)
	if err != nil {
		panic(err)
	}
	suite.namespace = cns.Name
	suite.wf = commands.InitWorkflowClient()
}

func (suite *E2ESuite) TearDownSuite() {
	deleteOptions := metav1.DeleteOptions{}
	err := suite.kube.CoreV1().Namespaces().Delete(suite.namespace, &deleteOptions)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Deleted namespace %s\n", suite.namespace)
}

func (suite *E2ESuite) Given() *Given {
	return &Given{suite: suite}
}

func getKubernetesClient() (*rest.Config, *kubernetes.Clientset) {
	if *kubeConfig == "" {
		panic("Kubeconfig not provided")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		panic(err)
	}

	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return config, clientSet
}
