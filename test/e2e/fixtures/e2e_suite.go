package fixtures

import (
	"bufio"
	"encoding/base64"
	"os"
	"strings"
	"time"

	// load the azure plugin (required to authenticate against AKS clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// load the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// load the oidc plugin (required to authenticate with OpenID Connect).
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/util/kubeconfig"
	"github.com/argoproj/argo/workflow/packer"
)

const Namespace = "argo"
const Label = "argo-e2e"

// Cron tests run in parallel, so use a different label so they are not deleted when a new test runs
const LabelCron = Label + "-cron"

var imageTag string
var k3d bool

func init() {
	output, err := runCli("git", "rev-parse", "--abbrev-ref=loose", "HEAD")
	if err != nil {
		panic(err)
	}
	imageTag = strings.TrimSpace(output)
	if imageTag == "master" {
		imageTag = "latest"
	}
	context, err := runCli("kubectl", "config", "current-context")
	if err != nil {
		panic(err)
	}
	k3d = strings.TrimSpace(context) == "k3s-default"
	log.WithFields(log.Fields{"imageTag": imageTag, "k3d": k3d}).Info()
}

type E2ESuite struct {
	suite.Suite
	Persistence       *Persistence
	RestConfig        *rest.Config
	wfClient          v1alpha1.WorkflowInterface
	wfTemplateClient  v1alpha1.WorkflowTemplateInterface
	cwfTemplateClient v1alpha1.ClusterWorkflowTemplateInterface
	cronClient        v1alpha1.CronWorkflowInterface
	KubeClient        kubernetes.Interface
	// Guard-rail.
	// A list of images that exist on the K3S node at the start of the test are probably those created as part
	// of the Kubernetes system (e.g. k8s.gcr.io/pause:3.1) or K3S. This is populated at the start of each test,
	// and checked at the end of each test.
	images map[string]bool
	// Guard-rail.
	// The number of archived workflows. If is changes between two tests, we have a problem.
	numWorkflows int
}

func (s *E2ESuite) SetupSuite() {
	var err error
	s.RestConfig, err = kubeconfig.DefaultRestConfig()
	s.CheckError(err)
	s.KubeClient, err = kubernetes.NewForConfig(s.RestConfig)
	s.CheckError(err)
	s.wfClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().Workflows(Namespace)
	s.wfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowTemplates(Namespace)
	s.cronClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().CronWorkflows(Namespace)
	s.Persistence = newPersistence(s.KubeClient)
	s.cwfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().ClusterWorkflowTemplates()
}

func (s *E2ESuite) listImages() map[string]bool {
	list, err := s.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	s.CheckError(err)
	images := make(map[string]bool)
	// looks O^3, but is actually going to be O(n)
	for _, node := range list.Items {
		for _, image := range node.Status.Images {
			for _, n := range image.Names {
				// We want to ignore hashes.
				if !strings.Contains(n, "@sha256") && strings.HasPrefix(n, "docker.io/") {
					images[n] = true
				}
			}
		}
	}
	return images
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
	s.images = s.listImages()
	s.importImages()
	numWorkflows := s.countWorkflows()
	if s.numWorkflows > 0 && s.numWorkflows != numWorkflows {
		s.T().Fatal("there should almost never be a change to the number of workflows between tests, this means the last test (not the current test) is bad and needs fixing - note this guard-rail does not work across test suites")
	}
	s.numWorkflows = numWorkflows
}

func (s *E2ESuite) countWorkflows() int {
	workflows, err := s.wfClient.List(metav1.ListOptions{})
	s.CheckError(err)
	return len(workflows.Items)
}

func (s *E2ESuite) importImages() {
	// If we are running K3D we should re-import these prior to running tests, as they may have been evicted.
	if k3d {
		for _, n := range []string{"docker.io/argoproj/argoexec:" + imageTag, "docker.io/library/cowsay:v1"} {
			if !s.images[n] {
				_, err := runCli("k3d", "import-images", n)
				s.CheckError(err)
			}
		}
	}
}

func (s *E2ESuite) DeleteResources(label string) {
	// delete all cron workflows
	cronList, err := s.cronClient.List(metav1.ListOptions{LabelSelector: label})
	s.CheckError(err)
	for _, cronWf := range cronList.Items {
		log.WithFields(log.Fields{"cronWorkflow": cronWf.Name}).Debug("Deleting cron workflow")
		err = s.cronClient.Delete(cronWf.Name, nil)
		s.CheckError(err)
	}

	// It is possible for a pod to become orphaned. This means that it's parent workflow
	// (as set in the  "workflows.argoproj.io/workflow" label) does not exist.
	// We need to delete orphans as well as test pods.
	// Get a list of all workflows.
	// if absent from this this it has been delete - so any associated pods are orphaned
	// if in the list it is either a test wf or not
	isTestWf := make(map[string]bool)
	{
		list, err := s.wfClient.List(metav1.ListOptions{LabelSelector: label})
		s.CheckError(err)
		for _, wf := range list.Items {
			isTestWf[wf.Name] = false
			if s.Persistence.IsEnabled() && wf.Status.IsOffloadNodeStatus() {
				err := s.Persistence.offloadNodeStatusRepo.Delete(string(wf.UID), wf.Status.OffloadNodeStatusVersion)
				s.CheckError(err)
			}
		}
	}

	// delete from the archive
	{
		if s.Persistence.IsEnabled() {
			archive := s.Persistence.workflowArchive
			parse, err := labels.ParseToRequirements(Label)
			s.CheckError(err)
			workflows, err := archive.ListWorkflows(Namespace, time.Time{}, time.Time{}, parse, 0, 0)
			s.CheckError(err)
			for _, workflow := range workflows {
				err := archive.DeleteWorkflow(string(workflow.UID))
				s.CheckError(err)
			}
		}
	}

	// delete all workflows
	{
		list, err := s.wfClient.List(metav1.ListOptions{LabelSelector: Label})
		s.CheckError(err)
		for _, wf := range list.Items {
			logCtx := log.WithFields(log.Fields{"workflow": wf.Name})
			logCtx.Debug("Deleting workflow")
			err = s.wfClient.Delete(wf.Name, &metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				continue
			}
			s.CheckError(err)
			isTestWf[wf.Name] = true
			for {
				_, err := s.wfClient.Get(wf.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					break
				}
				logCtx.Debug("Waiting for workflow to be deleted")
				time.Sleep(1 * time.Second)
			}
		}
	}

	// delete workflow pods
	{
		podInterface := s.KubeClient.CoreV1().Pods(Namespace)
		// it seems "argo delete" can leave pods behind
		pods, err := podInterface.List(metav1.ListOptions{LabelSelector: "workflows.argoproj.io/workflow"})
		s.CheckError(err)
		for _, pod := range pods.Items {
			workflow := pod.GetLabels()["workflows.argoproj.io/workflow"]
			testPod, owned := isTestWf[workflow]
			if testPod || !owned {
				logCtx := log.WithFields(log.Fields{"workflow": workflow, "podName": pod.Name, "testPod": testPod, "owned": owned})
				logCtx.Debug("Deleting pod")
				err := podInterface.Delete(pod.Name, nil)
				if !errors.IsNotFound(err) {
					s.CheckError(err)
				}
				for {
					_, err := podInterface.Get(pod.Name, metav1.GetOptions{})
					if errors.IsNotFound(err) {
						break
					}
					logCtx.Debug("Waiting for pod to be deleted")
					time.Sleep(1 * time.Second)
				}
			}
		}
	}

	// delete all workflow templates
	wfTmpl, err := s.wfTemplateClient.List(metav1.ListOptions{LabelSelector: label})
	s.CheckError(err)

	for _, wfTmpl := range wfTmpl.Items {
		log.WithField("template", wfTmpl.Name).Debug("Deleting workflow template")
		err = s.wfTemplateClient.Delete(wfTmpl.Name, nil)
		s.CheckError(err)
	}

	// delete all cluster workflow templates
	cwfTmpl, err := s.cwfTemplateClient.List(metav1.ListOptions{LabelSelector: label})
	s.CheckError(err)
	for _, cwfTmpl := range cwfTmpl.Items {
		log.WithField("template", cwfTmpl.Name).Debug("Deleting cluster workflow template")
		err = s.cwfTemplateClient.Delete(cwfTmpl.Name, nil)
		s.CheckError(err)
	}

	// Delete all resourcequotas
	rqList, err := s.KubeClient.CoreV1().ResourceQuotas(Namespace).List(metav1.ListOptions{LabelSelector: label})
	s.CheckError(err)
	for _, rq := range rqList.Items {
		log.WithField("resourcequota", rq.Name).Debug("Deleting resource quota")
		err = s.KubeClient.CoreV1().ResourceQuotas(Namespace).Delete(rq.Name, nil)
		s.CheckError(err)
	}
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
	// Using an arbitrary image will result in slow and flakey tests as we can't really predict when they'll be
	// downloaded or evicted. To keep tests fast and reliable you must use whitelisted images.
	imageWhitelist := map[string]bool{
		"docker.io/argoproj/argoexec:" + imageTag: true,
		"docker.io/library/cowsay:v1":             true,
		"docker.io/library/python:alpine3.6":      true,
	}
	for n := range s.listImages() {
		if !s.images[n] && !imageWhitelist[n] {
			s.T().Fatalf("non-whitelisted image used in test: %s", n)
		}
	}
	err = file.Close()
	s.CheckError(err)
}

func (s *E2ESuite) printWorkflowDiagnostics(name string) {
	logCtx := log.WithFields(log.Fields{"test": s.T().Name(), "workflow": name})
	// print logs
	wf, err := s.wfClient.Get(name, metav1.GetOptions{})
	s.CheckError(err)
	err = packer.DecompressWorkflow(wf)
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
		t:                     s.T(),
		client:                s.wfClient,
		wfTemplateClient:      s.wfTemplateClient,
		cwfTemplateClient:     s.cwfTemplateClient,
		cronClient:            s.cronClient,
		offloadNodeStatusRepo: s.Persistence.offloadNodeStatusRepo,
		kubeClient:            s.KubeClient,
	}
}
