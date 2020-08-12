package fixtures

import (
	"bufio"
	"encoding/base64"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	// load the azure plugin (required to authenticate against AKS clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// load the oidc plugin (required to authenticate with OpenID Connect).
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/util/kubeconfig"
	"github.com/argoproj/argo/workflow/hydrator"
)

const Namespace = "argo"
const Label = "argo-e2e"

// Cron tests run in parallel, so use a different label so they are not deleted when a new test runs
const LabelCron = Label + "-cron"

type E2ESuite struct {
	suite.Suite
	Persistence       *Persistence
	RestConfig        *rest.Config
	wfClient          v1alpha1.WorkflowInterface
	wfebClient        v1alpha1.WorkflowEventBindingInterface
	wfTemplateClient  v1alpha1.WorkflowTemplateInterface
	cwfTemplateClient v1alpha1.ClusterWorkflowTemplateInterface
	cronClient        v1alpha1.CronWorkflowInterface
	KubeClient        kubernetes.Interface
	hydrator          hydrator.Interface
}

func (s *E2ESuite) SetupSuite() {
	var err error
	s.RestConfig, err = kubeconfig.DefaultRestConfig()
	s.CheckError(err)
	s.KubeClient, err = kubernetes.NewForConfig(s.RestConfig)
	s.CheckError(err)
	s.wfClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().Workflows(Namespace)
	s.wfebClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowEventBindings(Namespace)
	s.wfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowTemplates(Namespace)
	s.cronClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().CronWorkflows(Namespace)
	s.Persistence = newPersistence(s.KubeClient)
	s.hydrator = hydrator.New(s.Persistence.offloadNodeStatusRepo)
	s.cwfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().ClusterWorkflowTemplates()
}

func (s *E2ESuite) TearDownSuite() {
	s.Persistence.Close()
}

func (s *E2ESuite) BeforeTest(suiteName, testName string) {
	dir := "/tmp/log/argo-e2e"
	err := os.MkdirAll(dir, 0777)
	s.CheckError(err)
	name := dir + "/" + suiteName + "-" + testName + ".log"
	f, err := os.Create(name)
	s.CheckError(err)
	err = file.setFile(f)
	s.CheckError(err)
	log.Infof("logging debug diagnostics to file://%s", name)
	s.DeleteResources(Label)
}

func (s *E2ESuite) countWorkflows() int {
	workflows, err := s.wfClient.List(metav1.ListOptions{})
	s.CheckError(err)
	return len(workflows.Items)
}


var foreground = metav1.DeletePropagationForeground
var foregroundDelete = &metav1.DeleteOptions{PropagationPolicy: &foreground}

func (s *E2ESuite) DeleteResources(label string) {

	hasTestLabel := metav1.ListOptions{LabelSelector: label}

	err := s.cronClient.DeleteCollection(foregroundDelete, hasTestLabel)
	s.CheckError(err)

	err = s.wfebClient.DeleteCollection(foregroundDelete, hasTestLabel)
	s.CheckError(err)

	err = s.wfClient.DeleteCollection(foregroundDelete, hasTestLabel)
	s.CheckError(err)

	// delete archived workflows from the archive
	if s.Persistence.IsEnabled() {
		archive := s.Persistence.workflowArchive
		parse, err := labels.ParseToRequirements(label)
		s.CheckError(err)
		workflows, err := archive.ListWorkflows(Namespace, time.Time{}, time.Time{}, parse, 0, 0)
		s.CheckError(err)
		for _, workflow := range workflows {
			err := archive.DeleteWorkflow(string(workflow.UID))
			s.CheckError(err)
		}
	}

	err = s.wfTemplateClient.DeleteCollection(foregroundDelete, hasTestLabel)
	s.CheckError(err)

	err = s.cwfTemplateClient.DeleteCollection(foregroundDelete, hasTestLabel)
	s.CheckError(err)

	err = s.KubeClient.CoreV1().ResourceQuotas(Namespace).DeleteCollection(foregroundDelete, hasTestLabel)
	s.CheckError(err)

	err = s.KubeClient.CoreV1().ConfigMaps(Namespace).DeleteCollection(foregroundDelete, hasTestLabel)
	s.CheckError(err)
}

func (s *E2ESuite) CheckError(err error) {
	s.T().Helper()
	if err != nil {
		s.T().Fatal(err)
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

func (s *E2ESuite) Run(name string, subtest func()) {
	// This add demarcation to the logs making it easier to differentiate the output of different tests.
	longName := s.T().Name() + "/" + name
	log.Debug("=== RUN " + longName)
	defer func() {
		if s.T().Failed() {
			log.Debug("=== FAIL " + longName)
			s.T().FailNow()
		} else if s.T().Skipped() {
			log.Debug("=== SKIP " + longName)
		} else {
			log.Debug("=== PASS " + longName)
		}
	}()
	s.Suite.Run(name, subtest)
}

func (s *E2ESuite) AfterTest(_, _ string) {
	wfs, err := s.wfClient.List(metav1.ListOptions{FieldSelector: "metadata.namespace=" + Namespace, LabelSelector: Label})
	s.CheckError(err)
	for _, wf := range wfs.Items {
		s.printWorkflowDiagnostics(wf.GetName())
	}
	err = file.Close()
	s.CheckError(err)
	s.DeleteResources(Label)
}

func (s *E2ESuite) printWorkflowDiagnostics(name string) {
	logCtx := log.WithFields(log.Fields{"test": s.T().Name(), "workflow": name})
	// print logs
	wf, err := s.wfClient.Get(name, metav1.GetOptions{})
	s.CheckError(err)
	err = s.hydrator.Hydrate(wf)
	s.CheckError(err)
	if wf.Status.IsOffloadNodeStatus() {
		offloaded, err := s.Persistence.offloadNodeStatusRepo.Get(string(wf.UID), wf.Status.OffloadNodeStatusVersion)
		s.CheckError(err)
		wf.Status.Nodes = offloaded
	}
	logCtx.Debug("Workflow metadata:")
	s.printJSON(wf.ObjectMeta)
	logCtx.Debug("Workflow status:")
	s.printJSON(wf.Status)
	for _, node := range wf.Status.Nodes {
		if node.Type != "Pod" {
			continue
		}
		logCtx := logCtx.WithFields(log.Fields{"node": node.DisplayName})
		s.printPodDiagnostics(logCtx, wf.Namespace, node.ID)
	}
}

func (s *E2ESuite) printJSON(obj interface{}) {
	// print status
	bytes, err := yaml.Marshal(obj)
	s.CheckError(err)
	log.Debug("---")
	for _, line := range strings.Split(string(bytes), "\n") {
		log.Debug("  " + line)
	}
	log.Debug("---")
}

func (s *E2ESuite) printPodDiagnostics(logCtx *log.Entry, namespace string, podName string) {
	logCtx = logCtx.WithFields(log.Fields{"pod": podName})
	pod, err := s.KubeClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		logCtx.Error("Cannot get pod")
		return
	}
	logCtx.Debug("Pod manifest:")
	s.printJSON(pod)
	containers := append(pod.Spec.InitContainers, pod.Spec.Containers...)
	logCtx.WithField("numContainers", len(containers)).Debug()
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
	logCtx.Debug("Container logs:")
	scanner := bufio.NewScanner(stream)
	log.Debug("---")
	for scanner.Scan() {
		log.Debug("  " + scanner.Text())
	}
	log.Debug("---")
}

func (s *E2ESuite) Given() *Given {
	return &Given{
		t:                 s.T(),
		client:            s.wfClient,
		wfebClient:        s.wfebClient,
		wfTemplateClient:  s.wfTemplateClient,
		cwfTemplateClient: s.cwfTemplateClient,
		cronClient:        s.cronClient,
		hydrator:          s.hydrator,
		kubeClient:        s.KubeClient,
	}
}
