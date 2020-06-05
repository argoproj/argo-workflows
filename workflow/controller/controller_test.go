package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	authorizationv1 "k8s.io/api/authorization/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	wfextv "github.com/argoproj/argo/pkg/client/informers/externalversions"
	hydratorfake "github.com/argoproj/argo/workflow/hydrator/fake"
	"github.com/argoproj/argo/workflow/metrics"
)

var helloWorldWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metadata:
      annotations:
        annotationKey1: "annotationValue1"
        annotationKey2: "annotationValue2"
      labels:
        labelKey1: "labelValue1"
        labelKey2: "labelValue2"
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var testDefaultWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
  labels:
    foo: bar
spec:
  entrypoint: whalesay
  serviceAccountName: whalesay
  templates:
  - name: whalesay
    metadata:
      annotations:
        annotationKey1: "annotationValue1"
        annotationKey2: "annotationValue2"
      labels:
        labelKey1: "labelValue1"
        labelKey2: "labelValue2"
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var testDefaultWfTTL = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  serviceAccountName: whalesay
  ttlSecondsAfterFinished: 7
  ttlStrategy:
    secondsAfterCompletion: 5
  templates:
  - name: whalesay
    metadata:
      annotations:
        annotationKey1: "annotationValue1"
        annotationKey2: "annotationValue2"
      labels:
        labelKey1: "labelValue1"
        labelKey2: "labelValue2"
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func newController(objects ...runtime.Object) (context.CancelFunc, *WorkflowController) {
	wfclientset := fakewfclientset.NewSimpleClientset(objects...)
	informerFactory := wfextv.NewSharedInformerFactory(wfclientset, 10*time.Minute)
	wftmplInformer := informerFactory.Argoproj().V1alpha1().WorkflowTemplates()
	cwftmplInformer := informerFactory.Argoproj().V1alpha1().ClusterWorkflowTemplates()
	ctx, cancel := context.WithCancel(context.Background())
	go wftmplInformer.Informer().Run(ctx.Done())
	go cwftmplInformer.Informer().Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), wftmplInformer.Informer().HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	if !cache.WaitForCacheSync(ctx.Done(), cwftmplInformer.Informer().HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	kube := fake.NewSimpleClientset()
	controller := &WorkflowController{
		Config: config.Config{
			ExecutorImage: "executor:latest",
		},
		kubeclientset:   kube,
		wfclientset:     wfclientset,
		completedPods:   make(chan string, 512),
		wftmplInformer:  wftmplInformer,
		cwftmplInformer: cwftmplInformer,
		wfQueue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		wfArchive:       sqldb.NullWorkflowArchive,
		hydrator:        hydratorfake.Noop,
		metrics:         metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{}),
	}
	return cancel, controller
}

func newControllerWithDefaults() (context.CancelFunc, *WorkflowController) {
	cancel, controller := newController()
	myBool := true
	controller.Config.WorkflowDefaults = &wfv1.Workflow{
		Spec: wfv1.WorkflowSpec{
			HostNetwork: &myBool,
		},
	}
	return cancel, controller
}

func newControllerWithComplexDefaults() (context.CancelFunc, *WorkflowController) {
	cancel, controller := newController()
	myBool := true
	var ten int32 = 10
	var seven int32 = 10
	controller.Config.WorkflowDefaults = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"annotation": "value",
			},
			Labels: map[string]string{
				"label": "value",
			},
		},
		Spec: wfv1.WorkflowSpec{
			HostNetwork:        &myBool,
			Entrypoint:         "good_entrypoint",
			ServiceAccountName: "my_service_account",
			TTLStrategy: &wfv1.TTLStrategy{
				SecondsAfterCompletion: &ten,
				SecondsAfterSuccess:    &ten,
				SecondsAfterFailure:    &ten,
			},
			TTLSecondsAfterFinished: &seven,
		},
	}
	return cancel, controller
}

func unmarshalWF(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

func unmarshalWFTmpl(yamlStr string) *wfv1.WorkflowTemplate {
	var wftmpl wfv1.WorkflowTemplate
	err := yaml.Unmarshal([]byte(yamlStr), &wftmpl)
	if err != nil {
		panic(err)
	}
	return &wftmpl
}

func unmarshalCWFTmpl(yamlStr string) *wfv1.ClusterWorkflowTemplate {
	var cwftmpl wfv1.ClusterWorkflowTemplate
	err := yaml.Unmarshal([]byte(yamlStr), &cwftmpl)
	if err != nil {
		panic(err)
	}
	return &cwftmpl
}

// makePodsPhase acts like a pod controller and simulates the transition of pods transitioning into a specified state
func makePodsPhase(t *testing.T, phase apiv1.PodPhase, kubeclientset kubernetes.Interface, namespace string) {
	podcs := kubeclientset.CoreV1().Pods(namespace)
	pods, err := podcs.List(metav1.ListOptions{})
	assert.NoError(t, err)
	for _, pod := range pods.Items {
		if pod.Status.Phase == "" {
			pod.Status.Phase = phase
			if phase == apiv1.PodFailed {
				pod.Status.Message = "Pod failed"
			}
			_, _ = podcs.Update(&pod)
		}
	}
}

// makePodsPhase acts like a pod controller and simulates the transition of pods transitioning into a specified state
func makePodsPhaseAll(t *testing.T, phase apiv1.PodPhase, kubeclientset kubernetes.Interface, namespace string) {
	podcs := kubeclientset.CoreV1().Pods(namespace)
	pods, err := podcs.List(metav1.ListOptions{})
	assert.NoError(t, err)
	for _, pod := range pods.Items {
		pod.Status.Phase = phase
		if phase == apiv1.PodFailed {
			pod.Status.Message = "Pod failed"
		}
		_, _ = podcs.Update(&pod)
	}
}

func TestAddingWorkflowDefaultValueIfValueNotExist(t *testing.T) {
	ans := true
	t.Run("WithoutDefaults", func(t *testing.T) {
		cancel, controller := newController()
		defer cancel()
		workflow := unmarshalWF(helloWorldWf)
		err := controller.setWorkflowDefaults(workflow)
		assert.NoError(t, err)
		assert.Equal(t, workflow, unmarshalWF(helloWorldWf))
	})
	t.Run("WithDefaults", func(t *testing.T) {
		cancel, controller := newControllerWithDefaults()
		defer cancel()
		defaultWorkflowSpec := unmarshalWF(helloWorldWf)
		err := controller.setWorkflowDefaults(defaultWorkflowSpec)
		assert.NoError(t, err)
		assert.Equal(t, defaultWorkflowSpec.Spec.HostNetwork, &ans)
		assert.NotEqual(t, defaultWorkflowSpec, unmarshalWF(helloWorldWf))
		assert.Equal(t, *defaultWorkflowSpec.Spec.HostNetwork, true)
	})
}

func TestAddingWorkflowDefaultComplex(t *testing.T) {
	cancel, controller := newControllerWithComplexDefaults()
	defer cancel()
	workflow := unmarshalWF(testDefaultWf)
	var ten int32 = 10
	assert.Equal(t, workflow.Spec.Entrypoint, "whalesay")
	assert.Nil(t, workflow.Spec.TTLStrategy)
	assert.Contains(t, workflow.Labels, "foo")
	err := controller.setWorkflowDefaults(workflow)
	assert.NoError(t, err)
	assert.NotEqual(t, workflow, unmarshalWF(testDefaultWf))
	assert.Equal(t, workflow.Spec.Entrypoint, "whalesay")
	assert.Equal(t, workflow.Spec.ServiceAccountName, "whalesay")
	assert.Equal(t, *workflow.Spec.TTLStrategy.SecondsAfterFailure, ten)
	assert.Contains(t, workflow.Labels, "foo")
	assert.Contains(t, workflow.Labels, "label")
	assert.Contains(t, workflow.Annotations, "annotation")
}

func TestAddingWorkflowDefaultComplexTwo(t *testing.T) {
	cancel, controller := newControllerWithComplexDefaults()
	defer cancel()
	workflow := unmarshalWF(testDefaultWfTTL)
	var ten int32 = 10
	var seven int32 = 7
	var five int32 = 5
	err := controller.setWorkflowDefaults(workflow)
	assert.NoError(t, err)
	assert.NotEqual(t, workflow, unmarshalWF(testDefaultWfTTL))
	assert.Equal(t, workflow.Spec.Entrypoint, "whalesay")
	assert.Equal(t, workflow.Spec.ServiceAccountName, "whalesay")
	assert.Equal(t, *workflow.Spec.TTLStrategy.SecondsAfterCompletion, five)
	assert.Equal(t, *workflow.Spec.TTLStrategy.SecondsAfterFailure, ten)
	assert.Equal(t, *workflow.Spec.TTLSecondsAfterFinished, seven)
	assert.NotContains(t, workflow.Labels, "foo")
	assert.Contains(t, workflow.Labels, "label")
	assert.Contains(t, workflow.Annotations, "annotation")
}

func TestNamespacedController(t *testing.T) {
	kubeClient := fake.Clientset{}
	allowed := false
	kubeClient.AddReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed},
		}, nil
	})

	_, controller := newController()
	controller.kubeclientset = kubernetes.Interface(&kubeClient)
	controller.cwftmplInformer = nil
	controller.createClusterWorkflowTemplateInformer(context.TODO())
	assert.Nil(t, controller.cwftmplInformer)
}

func TestClusterController(t *testing.T) {
	kubeClient := fake.Clientset{}
	allowed := true
	kubeClient.AddReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed},
		}, nil
	})

	_, controller := newController()
	controller.kubeclientset = kubernetes.Interface(&kubeClient)
	controller.cwftmplInformer = nil
	controller.createClusterWorkflowTemplateInformer(context.TODO())
	assert.NotNil(t, controller.cwftmplInformer)
}

func TestWorkflowController_archivedWorkflowGarbageCollector(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	controller.archivedWorkflowGarbageCollector(make(chan struct{}))
}

const wfWithTmplRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
  namespace: default
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
    - name: message
      value: "test"
  workflowTemplateRef:
    name: workflow-template-whalesay-template
`
const wfTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template
  namespace: default
spec:
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestCheckAndInitWorkflowTmplRef(t *testing.T) {
	//_, controller := newController()
	wf := unmarshalWF(wfWithTmplRef)
	wftmpl := unmarshalWFTmpl(wfTmpl)
	_, controller := newController(wf, wftmpl)
	//_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wftmpl)
	//assert.NoError(t, err)
	woc := wfOperationCtx{controller: controller,
		wf: wf}
	t.Run("WithWorkflowTmplRef", func(t *testing.T) {
		_, _, err := woc.loadExecutionSpec()
		assert.NoError(t, err)
		assert.Equal(t, &wftmpl.Spec.WorkflowSpec, woc.wfSpec)

	})
}
