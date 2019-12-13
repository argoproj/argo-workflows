package fixtures

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
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
	// delete all workflows
	log.WithFields(log.Fields{"test": s.T().Name()}).Info("Deleting all existing workflows")
	err = s.client.DeleteCollection(nil, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	// wait for all pods to be deleted
	for {
		log.WithFields(log.Fields{"test": s.T().Name()}).Info("Waiting for pods to go away")
		time.Sleep(1 * time.Second)
		pods, err := s.kubeClient.CoreV1().Pods("argo").List(metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		if len(pods.Items) <= 3 {
			break
		}
	}
}

func (s *E2ESuite) AfterTest(_, _ string) {
	// TODO - only on failure?
	if s.T().Failed() || true {
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
			if node.Type != "Pod" {
				continue
			}
			pods := s.kubeClient.CoreV1().Pods(wf.Namespace)
			podName := node.ID
			pod, err := pods.Get(podName, metav1.GetOptions{})
			logCtx := log.WithFields(log.Fields{"test": s.T().Name(), "wf": wf.Name, "node": node.DisplayName, "pod": podName})
			if err != nil {
				logCtx.Error("Cannot get pod")
				continue
			}
			for _, container := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
				logCtx = logCtx.WithFields(log.Fields{"container": container.Name, "image": container.Image})
				stream, err := pods.GetLogs(podName, &v1.PodLogOptions{Container: container.Name}).Stream()
				if err != nil {
					logCtx.WithField("err", err).Error("Cannot get logs")
					continue
				}
				logCtx.Info("Container logs:")
				scanner := bufio.NewScanner(stream)
				fmt.Println("---")
				for scanner.Scan() {
					fmt.Println("  " + scanner.Text())
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
