package controller

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	wfextv "github.com/argoproj/argo/pkg/client/informers/externalversions"
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

func newController() *WorkflowController {
	wfclientset := fakewfclientset.NewSimpleClientset()
	informerFactory := wfextv.NewSharedInformerFactory(wfclientset, 10*time.Minute)
	wftmplInformer := informerFactory.Argoproj().V1alpha1().WorkflowTemplates()
	cwftmplInformer := informerFactory.Argoproj().V1alpha1().ClusterWorkflowTemplates()
	ctx := context.Background()
	go wftmplInformer.Informer().Run(ctx.Done())
	go cwftmplInformer.Informer().Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), wftmplInformer.Informer().HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	if !cache.WaitForCacheSync(ctx.Done(), cwftmplInformer.Informer().HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	return &WorkflowController{
		Config: config.Config{
			ExecutorImage: "executor:latest",
		},
		kubeclientset:   fake.NewSimpleClientset(),
		wfclientset:     wfclientset,
		completedPods:   make(chan string, 512),
		wftmplInformer:  wftmplInformer,
		cwftmplInformer: cwftmplInformer,
		wfQueue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		wfArchive:       sqldb.NullWorkflowArchive,
		Metrics:         make(map[string]prometheus.Metric),
	}
}

func newControllerWithDefaults() *WorkflowController {
	wfclientset := fakewfclientset.NewSimpleClientset()
	informerFactory := wfextv.NewSharedInformerFactory(wfclientset, 10*time.Minute)
	wftmplInformer := informerFactory.Argoproj().V1alpha1().WorkflowTemplates()
	ctx := context.Background()
	go wftmplInformer.Informer().Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), wftmplInformer.Informer().HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	myBool := true
	return &WorkflowController{
		Config: config.Config{
			ExecutorImage: "executor:latest",
			WorkflowDefaults: &wfv1.Workflow{
				Spec: wfv1.WorkflowSpec{
					HostNetwork: &myBool,
				},
			},
		},
		kubeclientset:  fake.NewSimpleClientset(),
		wfclientset:    wfclientset,
		completedPods:  make(chan string, 512),
		wftmplInformer: wftmplInformer,
		wfQueue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		wfArchive:      sqldb.NullWorkflowArchive,
	}
}

func newControllerWithComplexDefaults() *WorkflowController {
	wfclientset := fakewfclientset.NewSimpleClientset()
	informerFactory := wfextv.NewSharedInformerFactory(wfclientset, 10*time.Minute)
	wftmplInformer := informerFactory.Argoproj().V1alpha1().WorkflowTemplates()
	ctx := context.Background()
	go wftmplInformer.Informer().Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), wftmplInformer.Informer().HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	myBool := true
	var ten int32 = 10
	var seven int32 = 10
	return &WorkflowController{
		Config: config.Config{
			ExecutorImage: "executor:latest",
			WorkflowDefaults: &wfv1.Workflow{
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
			},
		},
		kubeclientset:  fake.NewSimpleClientset(),
		wfclientset:    wfclientset,
		completedPods:  make(chan string, 512),
		wftmplInformer: wftmplInformer,
		wfQueue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		wfArchive:      sqldb.NullWorkflowArchive,
	}
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
	controller := newController()
	workflow := unmarshalWF(helloWorldWf)
	err := controller.setWorkflowDefaults(workflow)
	assert.NoError(t, err)
	assert.Equal(t, workflow, unmarshalWF(helloWorldWf))
	controllerDefaults := newControllerWithDefaults()
	defaultWorkflowSpec := unmarshalWF(helloWorldWf)
	err = controllerDefaults.setWorkflowDefaults(defaultWorkflowSpec)
	assert.NoError(t, err)
	assert.Equal(t, defaultWorkflowSpec.Spec.HostNetwork, &ans)
	assert.NotEqual(t, defaultWorkflowSpec, unmarshalWF(helloWorldWf))
	assert.Equal(t, *defaultWorkflowSpec.Spec.HostNetwork, true)
}

func TestAddingWorkflowDefaultComplex(t *testing.T) {
	controller := newControllerWithComplexDefaults()
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
	controller := newControllerWithComplexDefaults()
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

const wfWithTmplRef =`
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
const wfTmpl =`
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template
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
func TestCheckAndInitWorkflowTmplRef(t *testing.T){
	 controller := newController()
	 wf := unmarshalWF(wfWithTmplRef)
	 wftmpl := unmarshalWFTmpl(wfTmpl)
	 _, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wftmpl)
	 assert.NoError(t, err)
	 woc := wfOperationCtx{controller:controller,
	 	wf:wf}
	 t.Run("WithWorkflowTmplRef", func(t *testing.T) {
		 woc.checkAndInitWorkflowTmplRef()
		 assert.True(t, woc.hasTopLevelWFTmplRef)
		 assert.Equal(t,wftmpl.Name, woc.topLevelWFTmplRef.GetName())
	 })

	t.Run("WithoutWorkflowTmplRef", func(t *testing.T) {
		woc.wf.Spec.WorkflowTemplateRef = nil
		woc.checkAndInitWorkflowTmplRef()
		assert.False(t, woc.hasTopLevelWFTmplRef)
		assert.Nil(t,woc.topLevelWFTmplRef)
	})
}