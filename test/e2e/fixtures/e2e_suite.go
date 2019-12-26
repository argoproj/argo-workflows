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
	"upper.io/db.v3/postgresql"

	"github.com/argoproj/argo/cmd/argo/commands"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

var kubeConfig = os.Getenv("KUBECONFIG")

const Namespace = "argo"
const label = "argo-e2e"

func init() {
	if kubeConfig == "" {
		kubeConfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	_ = commands.NewCommand()
}

type E2ESuite struct {
	suite.Suite
	wfClient   v1alpha1.WorkflowInterface
	KubeClient kubernetes.Interface
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
	s.KubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	s.wfClient = commands.InitWorkflowClient()
	// delete all workflows
	list, err := s.wfClient.List(metav1.ListOptions{LabelSelector: label})
	if err != nil {
		panic(err)
	}
	for _, wf := range list.Items {
		logCtx := log.WithFields(log.Fields{"test": s.T().Name(), "workflow": wf.Name})
		logCtx.Infof("Deleting workflow")
		err = s.wfClient.Delete(wf.Name, &metav1.DeleteOptions{})
		if err != nil {
			panic(err)
		}
		// wait for workflow pods to be deleted
		for {
			// it seems "argo delete" can leave pods behind
			options := metav1.ListOptions{LabelSelector: "workflows.argoproj.io/workflow=" + wf.Name}
			err := s.KubeClient.CoreV1().Pods(Namespace).DeleteCollection(nil, options)
			if err != nil {
				panic(err)
			}
			pods, err := s.KubeClient.CoreV1().Pods(Namespace).List(options)
			if err != nil {
				panic(err)
			}
			if len(pods.Items) == 0 {
				break
			}
			logCtx.WithField("num", len(pods.Items)).Info("Waiting for workflow pods to go away")
			time.Sleep(1 * time.Second)
		}
	}
	// create database collection
	db, err := postgresql.Open(postgresql.ConnectionURL{User: "postgres", Password: "password", Host: "localhost"})
	if err != nil {
		panic(err)
	}
	// delete everything from history
	_, err = db.DeleteFrom("argo_workflow_history").Exec()
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
	wfs, err := s.wfClient.List(metav1.ListOptions{FieldSelector: "metadata.namespace=" + Namespace})
	if err != nil {
		s.T().Fatal(err)
	}
	for _, wf := range wfs.Items {
		s.printWorkflowDiagnostics(wf)
	}
}

func (s *E2ESuite) printWorkflowDiagnostics(wf wfv1.Workflow) {
	logCtx := log.WithFields(log.Fields{"test": s.T().Name(), "workflow": wf.Name})
	logCtx.Info("Workflow status:")
	printJSON(wf.Status)
	// print logs
	workflow, err := s.wfClient.Get(wf.Name, metav1.GetOptions{})
	if err != nil {
		s.T().Fatal(err)
	}
	for _, node := range workflow.Status.Nodes {
		if node.Type != "Pod" {
			continue
		}
		logCtx := logCtx.WithFields(log.Fields{"node": node.DisplayName})
		s.printPodDiagnostics(logCtx, workflow.Namespace, node.ID)
	}
}

func printJSON(obj interface{}) {
	// print status
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		panic(err)
	}
	fmt.Println("---")
	fmt.Println(string(bytes))
	fmt.Println("---")
}

func (s *E2ESuite) printPodDiagnostics(logCtx *log.Entry, namespace string, podName string) {
	logCtx = logCtx.WithFields(log.Fields{"pod": podName})
	pod, err := s.KubeClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		logCtx.Error("Cannot get pod")
		return
	}
	logCtx.Info("Pod manifest:")
	printJSON(pod)
	containers := append(pod.Spec.InitContainers, pod.Spec.Containers...)
	logCtx.WithField("numContainers", len(containers)).Info()
	for _, container := range containers {
		logCtx = logCtx.WithFields(log.Fields{"container": container.Name, "image": container.Image, "pod": pod.Name})
		s.printPodLogs(logCtx, pod.Namespace, pod.Name, container.Name)
	}
}

func (s *E2ESuite) printPodLogs(logCtx *log.Entry, namespace, pod, container string) {
	stream, err := s.KubeClient.CoreV1().Pods(namespace).GetLogs(pod, &v1.PodLogOptions{Container: container}).Stream()
	if err != nil {
		logCtx.WithField("err", err).Error("Cannot get logs")
		return
	}
	defer func() { _ = stream.Close() }()
	logCtx.Info("Container logs:")
	scanner := bufio.NewScanner(stream)
	fmt.Println("---")
	for scanner.Scan() {
		fmt.Println("  " + scanner.Text())
	}
	fmt.Println("---")
}

func (s *E2ESuite) Given() *Given {
	return &Given{
		t:      s.T(),
		client: s.wfClient,
	}
}
