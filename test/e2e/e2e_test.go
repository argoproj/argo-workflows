package e2e

import (
	"flag"
	"fmt"
	"io/ioutil"
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
)

type E2ESuite struct {
	suite.Suite
	namespace string
}

func (suite *E2ESuite) SetupSuite() {
	if *kubeConfig == "" {
		suite.T().Skip("Skipping test. Kubeconfig not provided")
	}

	_, clientset := getKubernetesClient()
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "argo-e2e-test-",
		},
	}
	cns, err := clientset.CoreV1().Namespaces().Create(ns)
	if err != nil {
		panic(err)
	}
	suite.namespace = cns.Name
}

func (suite *E2ESuite) TearDownSuite() {
	_, clientset := getKubernetesClient()
	deleteOptions := metav1.DeleteOptions{}
	err := clientset.CoreV1().Namespaces().Delete(suite.namespace, &deleteOptions)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Deleted namespace %s\n", suite.namespace)
}

var kubeConfig = flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "Path to Kubernetes config file")

func init() {
	_ = commands.NewCommand()
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
func createTempFile(text string) (string, func()) {
	content := []byte(text)
	tmpfile, err := ioutil.TempFile("", "argo_test")
	if err != nil {
		panic(err)
	}
	if _, err := tmpfile.Write(content); err != nil {
		panic(err)
	}
	if err := tmpfile.Close(); err != nil {
		panic(err)
	}
	return tmpfile.Name(), func() { _ = os.Remove(tmpfile.Name()) }
}
