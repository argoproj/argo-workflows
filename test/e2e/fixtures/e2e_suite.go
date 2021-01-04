package fixtures

import (
	"encoding/base64"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/util/kubeconfig"
	"github.com/argoproj/argo/workflow/hydrator"
)

const Namespace = "argo"
const Label = "argo-e2e"
const defaultTimeout = 30 * time.Second

type E2ESuite struct {
	suite.Suite
	Config            *config.Config
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
	configController := config.NewController(Namespace, "workflow-controller-configmap", s.KubeClient, config.EmptyConfigFunc)
	c, err := configController.Get()
	s.CheckError(err)
	s.Config = c.(*config.Config)
	s.wfClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().Workflows(Namespace)
	s.wfebClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowEventBindings(Namespace)
	s.wfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowTemplates(Namespace)
	s.cronClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().CronWorkflows(Namespace)
	s.Persistence = newPersistence(s.KubeClient, s.Config)
	s.hydrator = hydrator.New(s.Persistence.offloadNodeStatusRepo)
	s.cwfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().ClusterWorkflowTemplates()
}

func (s *E2ESuite) TearDownSuite() {
	s.Persistence.Close()
}

func (s *E2ESuite) BeforeTest(string, string) {
	s.DeleteResources()
}

var foreground = metav1.DeletePropagationForeground
var background = metav1.DeletePropagationBackground

func (s *E2ESuite) DeleteResources() {

	hasTestLabel := metav1.ListOptions{LabelSelector: Label}
	resources := []schema.GroupVersionResource{
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.CronWorkflowPlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowPlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowTemplatePlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.ClusterWorkflowTemplatePlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowEventBindingPlural},
		{Version: "v1", Resource: "resourcequotas"},
		{Version: "v1", Resource: "configmaps"},
	}

	for _, r := range resources {
		err := s.dynamicFor(r).DeleteCollection(&metav1.DeleteOptions{PropagationPolicy: &background}, hasTestLabel)
		s.CheckError(err)
	}

	// delete archived workflows from the archive
	if s.Persistence.IsEnabled() {
		archive := s.Persistence.workflowArchive
		parse, err := labels.ParseToRequirements(Label)
		s.CheckError(err)
		workflows, err := archive.ListWorkflows(Namespace, time.Time{}, time.Time{}, parse, 0, 0)
		s.CheckError(err)
		for _, w := range workflows {
			err := archive.DeleteWorkflow(string(w.UID))
			s.CheckError(err)
		}
	}

	for _, r := range resources {
		for {
			list, err := s.dynamicFor(r).List(hasTestLabel)
			s.CheckError(err)
			if len(list.Items) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *E2ESuite) NeedsCI() {
	if os.Getenv("CI") != "true" {
		s.T().Skip("test needs CI")
	}
}

func (s *E2ESuite) NeedsOffloading() {
	if !s.Persistence.IsEnabled() {
		s.T().Skip("test needs offloading, but persistence not enabled")
	}
}

func (s *E2ESuite) dynamicFor(r schema.GroupVersionResource) dynamic.ResourceInterface {
	resourceInterface := dynamic.NewForConfigOrDie(s.RestConfig).Resource(r)
	if r.Resource == workflow.ClusterWorkflowTemplatePlural {
		return resourceInterface
	}
	return resourceInterface.Namespace(Namespace)
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
