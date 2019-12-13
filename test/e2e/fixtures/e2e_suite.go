package fixtures

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/argoproj/argo/cmd/argo/commands"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

var kubeConfig = os.Getenv("KUBECONFIG")

func init() {
	if kubeConfig == "" {
		kubeConfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	_ = commands.NewCommand()
}

type E2ESuite struct {
	suite.Suite
	client     v1alpha1.WorkflowInterface
	kubeClient kubernetes.Interface
}

func (s *E2ESuite) SetupSuite() {
	_, err := os.Stat(kubeConfig)
	if os.IsNotExist(err) {
		s.T().Skip("Skipping test: " + err.Error())
	}
}

func (s *E2ESuite) BeforeTest(_, _ string) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		panic(err)
	}
	s.kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	s.client = commands.InitWorkflowClient()
	log.WithField("test", s.T().Name()).Info("Deleting all existing workflows")
	timeout := int64(10)
	err = s.client.DeleteCollection(nil, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		panic(err)
	}
}

func (s *E2ESuite) AfterTest(_, _ string) {
	if s.T().Failed() {
		s.printDiagnostics()
	}
}

func (s *E2ESuite) printDiagnostics() {
	wfs, err := s.client.List(metav1.ListOptions{})
	if err != nil {
		s.T().Fatal(err)
	}
	for _, wf := range wfs.Items {
		log.WithFields(log.Fields{"test": s.T().Name(), "wf": wf.Name}).Info("Workflow status:")
		// print status
		bytes, err := yaml.Marshal(wf.Status)
		if err != nil {
			s.T().Fatal(err)
		}
		fmt.Println("---")
		fmt.Println(string(bytes))
		fmt.Println("---")
		// print logs
		wf, err := s.client.Get(wf.Name, metav1.GetOptions{})
		if err != nil {
			s.T().Fatal(err)
		}
		for _, node := range wf.Status.Nodes {
			pods := s.kubeClient.CoreV1().Pods(wf.Namespace)
			podName := node.ID
			pod, err := pods.Get(podName, metav1.GetOptions{})
			if apierr.IsNotFound(err) {
				log.WithFields(log.Fields{"test": s.T().Name(), "wf": wf.Name, "node": node.DisplayName, "pod": podName}).Warn("Not found")
				continue
			}
			if err != nil {
				s.T().Fatal(err)
			}
			for _, container := range pod.Status.ContainerStatuses {
				log.WithFields(log.Fields{"test": s.T().Name(), "wf": wf.Name, "node": node.DisplayName, "pod": podName, "container": container.Name, "state": container.State}).Info("Container logs:")
				if container.Started == nil {
					continue
				}
				stream, err := pods.GetLogs(podName, &v1.PodLogOptions{Container: container.Name,}).Stream()
				if err != nil {
					s.T().Fatal(err)
				}
				scanner := bufio.NewScanner(stream)
				fmt.Println("---")
				for scanner.Scan() {
					fmt.Println(scanner.Text())
				}
				fmt.Println("---")
				_ = stream.Close()
			}
		}
	}
}

func (s *E2ESuite) Given() *Given {
	return &Given{
		t:      s.T(),
		client: s.client,
	}
}
