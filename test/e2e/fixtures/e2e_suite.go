package fixtures

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/argo/commands"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/util/kubeconfig"
	"github.com/argoproj/argo/workflow/packer"
)

const Namespace = "argo"
const Label = "argo-e2e"

// Cron tests run in parallel, so use a different label so they are not deleted when a new test runs
const LabelCron = Label + "-cron"

func init() {
	_ = commands.NewCommand()
}

type E2ESuite struct {
	suite.Suite
	Diagnostics      *Diagnostics
	Persistence      *Persistence
	RestConfig       *rest.Config
	wfClient         v1alpha1.WorkflowInterface
	wfTemplateClient v1alpha1.WorkflowTemplateInterface
	cronClient       v1alpha1.CronWorkflowInterface
	KubeClient       kubernetes.Interface
}

func (s *E2ESuite) SetupSuite() {
	var err error
	s.RestConfig, err = kubeconfig.DefaultRestConfig()
	if err != nil {
		panic(err)
	}
	s.KubeClient, err = kubernetes.NewForConfig(s.RestConfig)
	if err != nil {
		panic(err)
	}

	s.wfClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().Workflows(Namespace)
	s.wfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowTemplates(Namespace)
	s.cronClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().CronWorkflows(Namespace)
	s.Persistence = newPersistence(s.KubeClient)
}

func (s *E2ESuite) TearDownSuite() {
	s.Persistence.Close()
}

func (s *E2ESuite) BeforeTest(_, _ string) {
	s.Diagnostics = &Diagnostics{}

	s.DeleteResources(Label)
}

func (s *E2ESuite) DeleteResources(label string) {
	// delete all cron workflows
	cronList, err := s.cronClient.List(metav1.ListOptions{LabelSelector: label})
	if err != nil {
		panic(err)
	}
	for _, cronWf := range cronList.Items {
		log.WithFields(log.Fields{"cronWorkflow": cronWf.Name}).Info("Deleting cron workflow")
		err = s.cronClient.Delete(cronWf.Name, nil)
		if err != nil {
			panic(err)
		}
	}

	// It is possible for a pod to become orphaned. This means that it's parent workflow
	// (as set in the  "workflows.argoproj.io/workflow" label) does not exist.
	// We need to delete orphans as well as test pods.
	// Get a list of all workflows.
	// if absent from this this it has been delete - so any associated pods are orphaned
	// if in the list it is either a test wf or not
	isTestWf := make(map[string]bool)
	{
		list, err := s.wfClient.List(metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, wf := range list.Items {
			isTestWf[wf.Name] = false
			if s.Persistence.IsEnabled() {
				// TODO - may make tests flakey
				err := s.Persistence.offloadNodeStatusRepo.Delete(string(wf.UID), wf.Status.OffloadNodeStatusVersion)
				if err != nil {
					panic(err)
				}
				err = s.Persistence.workflowArchive.DeleteWorkflow(string(wf.UID))
				if err != nil {
					panic(err)
				}
			}
		}
	}

	// delete all workflows
	{
		list, err := s.wfClient.List(metav1.ListOptions{LabelSelector: label})
		if err != nil {
			panic(err)
		}
		for _, wf := range list.Items {
			logCtx := log.WithFields(log.Fields{"workflow": wf.Name})
			logCtx.Infof("Deleting workflow")
			err = s.wfClient.Delete(wf.Name, &metav1.DeleteOptions{})
			if err != nil {
				panic(err)
			}
			isTestWf[wf.Name] = true
			for {
				_, err := s.wfClient.Get(wf.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					break
				}
				logCtx.Info("Waiting for workflow to be deleted")
				time.Sleep(1 * time.Second)
			}
		}
	}

	// delete workflow pods
	{
		podInterface := s.KubeClient.CoreV1().Pods(Namespace)
		// it seems "argo delete" can leave pods behind
		pods, err := podInterface.List(metav1.ListOptions{LabelSelector: "workflows.argoproj.io/workflow"})
		if err != nil {
			panic(err)
		}
		for _, pod := range pods.Items {
			workflow := pod.GetLabels()["workflows.argoproj.io/workflow"]
			testPod, owned := isTestWf[workflow]
			if testPod || !owned {
				logCtx := log.WithFields(log.Fields{"workflow": workflow, "podName": pod.Name, "testPod": testPod, "owned": owned})
				logCtx.Info("Deleting pod")
				err := podInterface.Delete(pod.Name, nil)
				if !errors.IsNotFound(err) {
					panic(err)
				}
				for {
					_, err := podInterface.Get(pod.Name, metav1.GetOptions{})
					if errors.IsNotFound(err) {
						break
					}
					logCtx.Info("Waiting for pod to be deleted")
					time.Sleep(1 * time.Second)
				}
			}
		}
	}

	// delete all workflow templates
	wfTmpl, err := s.wfTemplateClient.List(metav1.ListOptions{LabelSelector: label})
	if err != nil {
		panic(err)
	}
	for _, wfTmpl := range wfTmpl.Items {
		log.WithField("template", wfTmpl.Name).Info("Deleting workflow template")
		err = s.wfTemplateClient.Delete(wfTmpl.Name, nil)
		if err != nil {
			panic(err)
		}
	}
}

func (s *E2ESuite) GetBasicAuthToken() string {
	if s.RestConfig.Username == "" {
		return ""
	}
	auth := s.RestConfig.Username + ":" + s.RestConfig.Password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (s *E2ESuite) GetServiceAccountToken() (string, error) {
	// create the clientset
	clientset, err := kubernetes.NewForConfig(s.RestConfig)
	if err != nil {
		return "", err
	}
	secretList, err := clientset.CoreV1().Secrets("argo").List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, sec := range secretList.Items {
		if strings.HasPrefix(sec.Name, "argo-server-token") {
			return string(sec.Data["token"]), nil
		}
	}
	return "", nil
}

func (s *E2ESuite) AfterTest(_, _ string) {
	if s.T().Failed() {
		s.printDiagnostics()
	}
}

func (s *E2ESuite) printDiagnostics() {
	s.Diagnostics.Print()
	wfs, err := s.wfClient.List(metav1.ListOptions{FieldSelector: "metadata.namespace=" + Namespace, LabelSelector: Label})
	if err != nil {
		s.T().Fatal(err)
	}
	for _, wf := range wfs.Items {
		s.printWorkflowDiagnostics(wf)
	}
}

func (s *E2ESuite) printWorkflowDiagnostics(wf wfv1.Workflow) {
	logCtx := log.WithFields(log.Fields{"test": s.T().Name(), "workflow": wf.Name})
	logCtx.Info("Workflow metadata:")
	printJSON(wf.ObjectMeta)
	logCtx.Info("Workflow status:")
	printJSON(wf.Status)
	// print logs
	workflow, err := s.wfClient.Get(wf.Name, metav1.GetOptions{})
	if err != nil {
		s.T().Fatal(err)
	}
	err = packer.DecompressWorkflow(workflow)
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
		t:                     s.T(),
		diagnostics:           s.Diagnostics,
		client:                s.wfClient,
		wfTemplateClient:      s.wfTemplateClient,
		cronClient:            s.cronClient,
		offloadNodeStatusRepo: s.Persistence.offloadNodeStatusRepo,
		kubeClient:            s.KubeClient,
	}
}
