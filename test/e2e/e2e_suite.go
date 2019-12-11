package e2e

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/argoproj/argo/cmd/argo/commands"
)

var kubeConfig = flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "Path to Kubernetes config file")

const testingLabel = "e2e.argoproj.io"

func init() {
	_ = commands.NewCommand()
}

type E2ESuite struct {
	suite.Suite
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
	kube, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	namespaces := kube.CoreV1().Namespaces()
	list, err := namespaces.List(metav1.ListOptions{LabelSelector: testingLabel})
	if err != nil {
		panic(err)
	}
	for _, item := range list.Items {
		err := namespaces.Delete(item.Name, nil)
		if err != nil {
			panic(err)
		}
	}
	_, err = namespaces.Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "argo-e2e-test-", Labels: map[string]string{testingLabel: "true"}}})
	if err != nil {
		panic(err)
	}
}

func (suite *E2ESuite) Given() *Given {
	return &Given{suite: suite}
}
