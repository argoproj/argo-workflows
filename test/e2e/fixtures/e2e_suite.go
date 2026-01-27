package fixtures

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/secrets"

	apierr "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/types"

	"github.com/TwiN/go-color"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/kubeconfig"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

const (
	Namespace = "argo"
	Label     = workflow.WorkflowFullName + "/test"     // mark this workflow as a test
	Backfill  = workflow.WorkflowFullName + "/backfill" // clean backfill workflows
)

var timeoutBias = env.LookupEnvDurationOr(logging.InitLoggerInContext(), "E2E_WAIT_TIMEOUT_BIAS", 0*time.Second)

var defaultTimeout = env.LookupEnvDurationOr(logging.InitLoggerInContext(), "E2E_WAIT_TIMEOUT", 60*time.Second) + timeoutBias

var EnvFactor = env.LookupEnvIntOr(logging.InitLoggerInContext(), "E2E_ENV_FACTOR", 1)

type E2ESuite struct {
	suite.Suite
	Config            *config.Config
	Persistence       *Persistence
	RestConfig        *rest.Config
	wfClient          v1alpha1.WorkflowInterface
	wfebClient        v1alpha1.WorkflowEventBindingInterface
	wfTemplateClient  v1alpha1.WorkflowTemplateInterface
	wftsClient        v1alpha1.WorkflowTaskSetInterface
	cwfTemplateClient v1alpha1.ClusterWorkflowTemplateInterface
	cronClient        v1alpha1.CronWorkflowInterface
	KubeClient        kubernetes.Interface
	hydrator          hydrator.Interface
	testStartedAt     time.Time
	slowTests         []string
}

func (s *E2ESuite) SetupSuite() {
	var err error
	s.RestConfig, err = kubeconfig.DefaultRestConfig()
	s.CheckError(err)
	s.KubeClient, err = kubernetes.NewForConfig(s.RestConfig)
	s.CheckError(err)
	configController := config.NewController(Namespace, common.ConfigMapName, s.KubeClient)

	ctx := logging.TestContext(s.T().Context())
	c, err := configController.Get(ctx)
	s.CheckError(err)
	s.Config = c
	s.wfClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().Workflows(Namespace)
	s.wfebClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowEventBindings(Namespace)
	s.wfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowTemplates(Namespace)
	s.wftsClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().WorkflowTaskSets(Namespace)
	s.cronClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().CronWorkflows(Namespace)
	s.Persistence = NewPersistence(ctx, s.KubeClient, s.Config)
	s.hydrator = hydrator.New(s.Persistence.OffloadNodeStatusRepo)
	s.cwfTemplateClient = versioned.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().ClusterWorkflowTemplates()
}

func (s *E2ESuite) TearDownSuite() {
	s.Persistence.Close()
	for _, x := range s.slowTests {
		_, _ = fmt.Println(color.Ize(color.Yellow, fmt.Sprintf("=== SLOW TEST:  %s", x)))
	}
	if s.T().Failed() {
		s.T().Log("to learn how to diagnose failed tests: https://argo-workflows.readthedocs.io/en/latest/running-locally/#running-e2e-tests-locally")
	}
}

func (s *E2ESuite) BeforeTest(string, string) {
	start := time.Now()
	s.DeleteResources()
	if time.Since(start) > time.Second {
		_, _ = fmt.Printf("LONG SET-UP took %v (maybe previous test was slow)\n", time.Since(start).Truncate(time.Second))
	}
	s.testStartedAt = time.Now()
}

func (s *E2ESuite) AfterTest(suiteName, testName string) {
	if s.T().Skipped() { // by default, we don't get good logging at test end
		_, _ = fmt.Println(color.Ize(color.Gray, "=== SKIP: "+suiteName+"/"+testName))
	} else if s.T().Failed() { // by default, we don't get good logging at test end
		_, _ = fmt.Println(color.Ize(color.Red, "=== FAIL: "+suiteName+"/"+testName))
		os.Exit(1)
	} else {
		_, _ = fmt.Println(color.Ize(color.Green, "=== PASS: "+suiteName+"/"+testName))
		took := time.Since(s.testStartedAt)
		if took > 15*time.Second {
			s.slowTests = append(s.slowTests, fmt.Sprintf("%s/%s took %v", suiteName, testName, took.Truncate(time.Second)))
		}
	}
}

func (s *E2ESuite) DeleteResources() {
	ctx := logging.TestContext(s.T().Context())

	l := func(r schema.GroupVersionResource) string {
		if r.Resource == "pods" {
			return common.LabelKeyWorkflow
		}
		return Label
	}

	pods := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	resources := []schema.GroupVersionResource{
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.CronWorkflowPlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowPlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowTemplatePlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.ClusterWorkflowTemplatePlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowEventBindingPlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: "sensors"},
		{Group: workflow.Group, Version: workflow.Version, Resource: "eventsources"},
		pods,
		{Version: "v1", Resource: "resourcequotas"},
		{Version: "v1", Resource: "configmaps"},
	}
	for _, r := range resources {
		for {
			// remove finalizer from all the resources of the given GroupVersionResource
			resourceInf := DynamicFor(s.RestConfig, pods)
			resourceList, err := resourceInf.List(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyCompleted + "=false"})
			s.CheckError(err)
			for _, item := range resourceList.Items {
				patch, err := json.Marshal(map[string]interface{}{
					"metadata": map[string]interface{}{
						"finalizers": []string{},
					},
				})
				s.CheckError(err)
				_, err = resourceInf.Patch(ctx, item.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
				if err != nil && !apierr.IsNotFound(err) {
					s.CheckError(err)
				}
			}
			s.CheckError(DynamicFor(s.RestConfig, r).DeleteCollection(ctx, metav1.DeleteOptions{GracePeriodSeconds: ptr.To(int64(2))}, metav1.ListOptions{LabelSelector: l(r)}))
			ls, err := DynamicFor(s.RestConfig, r).List(ctx, metav1.ListOptions{LabelSelector: l(r)})
			s.CheckError(err)
			if len(ls.Items) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// delete archived workflows from the archive
	if s.Persistence.IsEnabled() {
		archive := s.Persistence.WorkflowArchive
		parse, err := labels.ParseToRequirements(Label)
		s.CheckError(err)
		workflows, err := archive.ListWorkflows(ctx, utils.ListOptions{
			Namespace:         Namespace,
			LabelRequirements: parse,
		})
		s.CheckError(err)
		for _, w := range workflows {
			err := archive.DeleteWorkflow(ctx, string(w.UID))
			s.CheckError(err)
		}
		parse, err = labels.ParseToRequirements(Backfill)
		s.CheckError(err)
		backfillWorkflows, err := archive.ListWorkflows(ctx, utils.ListOptions{
			Namespace:         Namespace,
			LabelRequirements: parse,
		})
		s.CheckError(err)
		for _, w := range backfillWorkflows {
			err := archive.DeleteWorkflow(ctx, string(w.UID))
			s.CheckError(err)
		}
	}
}

func (s *E2ESuite) CheckError(err error) {
	CheckError(s.T(), err)
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

	ctx := logging.TestContext(s.T().Context())
	sec, err := clientset.CoreV1().Secrets(Namespace).Get(ctx, secrets.TokenName("argo-server"), metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(sec.Data["token"]), nil
}

func (s *E2ESuite) Given() *Given {
	bearerToken, err := s.GetServiceAccountToken()
	if err != nil {
		s.T().Fatal(err)
	}
	return &Given{
		t:                 s.T(),
		client:            s.wfClient,
		wfebClient:        s.wfebClient,
		wfTemplateClient:  s.wfTemplateClient,
		wftsClient:        s.wftsClient,
		cwfTemplateClient: s.cwfTemplateClient,
		cronClient:        s.cronClient,
		hydrator:          s.hydrator,
		kubeClient:        s.KubeClient,
		bearerToken:       bearerToken,
		restConfig:        s.RestConfig,
		config:            s.Config,
	}
}
