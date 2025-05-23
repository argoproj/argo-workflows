package controller

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/argoproj/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/scheme"
	wfextv "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
	armocks "github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories/mocks"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v3/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/entrypoint"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/estimation"
	"github.com/argoproj/argo-workflows/v3/workflow/events"
	hydratorfake "github.com/argoproj/argo-workflows/v3/workflow/hydrator/fake"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
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

var fromExrpessingWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: from-expression
spec:
  entrypoint: main
  arguments:
    artifacts:
    - name: foo
      raw:
        data: |
          Hello
  templates:
    - name: main
      inputs:
        artifacts:
        - name: foo
      steps:
      - - name: hello
          inline:
            container:
              image: docker/whalesay:latest
      outputs:
        artifacts:
        - name: result
          fromExpression: "1 == 1 ? inputs.artifacts.foo : inputs.artifacts.foo"
`

var helloDaemonWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    daemon: true
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

var testDefaultVolumeClaimTemplateWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
  labels:
    foo: bar
spec:
  volumeClaimTemplates:
  - metadata:
      name: workdir
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 1Mi
      storageClassName: local-path
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

type testEventRecorderManager struct {
	eventRecorder *record.FakeRecorder
}

func (t testEventRecorderManager) Get(string) record.EventRecorder {
	return t.eventRecorder
}

var _ events.EventRecorderManager = &testEventRecorderManager{}

var defaultServiceAccount = &apiv1.ServiceAccount{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "default",
		Namespace: "default",
	},
	Secrets: []apiv1.ObjectReference{{}},
}

func newController(options ...interface{}) (context.CancelFunc, *WorkflowController) {
	// get all the objects and add to the fake
	var objects, coreObjects []runtime.Object
	for _, opt := range options {
		switch v := opt.(type) {
		case *apiv1.ServiceAccount:
			coreObjects = append(coreObjects, v)
		case runtime.Object:
			objects = append(objects, v)
		}
	}
	wfclientset := fakewfclientset.NewSimpleClientset(objects...)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme.Scheme, objects...)
	informerFactory := wfextv.NewSharedInformerFactory(wfclientset, 0)
	ctx, cancel := context.WithCancel(context.Background())
	kube := fake.NewSimpleClientset(coreObjects...)
	wfc := &WorkflowController{
		Config: config.Config{
			Images: map[string]config.Image{
				"my-image": {
					Entrypoint: []string{"my-entrypoint"},
					Cmd:        []string{"my-cmd"},
				},
				"argoproj/argosay:v2":    {Cmd: []string{""}},
				"docker/whalesay:latest": {Cmd: []string{""}},
				"busybox":                {Cmd: []string{""}},
			},
		},
		artifactRepositories: armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
			S3: &wfv1.S3ArtifactRepository{
				S3Bucket: wfv1.S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"},
			},
		}),
		cliExecutorLogFormat:      "text",
		kubeclientset:             kube,
		dynamicInterface:          dynamicClient,
		wfclientset:               wfclientset,
		workflowKeyLock:           sync.NewKeyLock(),
		wfArchive:                 sqldb.NullWorkflowArchive,
		hydrator:                  hydratorfake.Noop,
		estimatorFactory:          estimation.DummyEstimatorFactory,
		eventRecorderManager:      &testEventRecorderManager{eventRecorder: record.NewFakeRecorder(64)},
		archiveLabelSelector:      labels.Everything(),
		cacheFactory:              controllercache.NewCacheFactory(kube, "default"),
		progressPatchTickDuration: envutil.LookupEnvDurationOr(common.EnvVarProgressPatchTickDuration, 1*time.Minute),
		progressFileTickDuration:  envutil.LookupEnvDurationOr(common.EnvVarProgressFileTickDuration, 3*time.Second),
		maxStackDepth:             maxAllowedStackDepth,
	}

	for _, opt := range options {
		switch v := opt.(type) {
		// any post-processing
		case func(workflowController *WorkflowController):
			v(wfc)
		}
	}

	// always compare to NewWorkflowController to see what this block of code should be doing
	{
		wfc.metrics = metrics.New(metrics.ServerConfig{}, metrics.ServerConfig{})
		wfc.entrypoint = entrypoint.New(kube, wfc.Config.Images)
		wfc.wfQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
		wfc.throttler = wfc.newThrottler()
		wfc.podCleanupQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
		wfc.rateLimiter = wfc.newRateLimiter()
	}

	// always compare to WorkflowController.Run to see what this block of code should be doing
	{
		wfc.wfInformer = util.NewWorkflowInformer(dynamicClient, "", 0, wfc.tweakListRequestListOptions, wfc.tweakWatchRequestListOptions, indexers)
		wfc.wfTaskSetInformer = informerFactory.Argoproj().V1alpha1().WorkflowTaskSets()
		wfc.artGCTaskInformer = informerFactory.Argoproj().V1alpha1().WorkflowArtifactGCTasks()
		wfc.taskResultInformer = wfc.newWorkflowTaskResultInformer()
		wfc.wftmplInformer = informerFactory.Argoproj().V1alpha1().WorkflowTemplates()
		wfc.addWorkflowInformerHandlers(ctx)
		wfc.podInformer = wfc.newPodInformer(ctx)
		wfc.configMapInformer = wfc.newConfigMapInformer()
		wfc.createSynchronizationManager(ctx)
		_ = wfc.initManagers(ctx)

		go wfc.wfInformer.Run(ctx.Done())
		go wfc.wftmplInformer.Informer().Run(ctx.Done())
		go wfc.podInformer.Run(ctx.Done())
		go wfc.wfTaskSetInformer.Informer().Run(ctx.Done())
		go wfc.artGCTaskInformer.Informer().Run(ctx.Done())
		go wfc.taskResultInformer.Run(ctx.Done())
		wfc.cwftmplInformer = informerFactory.Argoproj().V1alpha1().ClusterWorkflowTemplates()
		go wfc.cwftmplInformer.Informer().Run(ctx.Done())
		// wfc.waitForCacheSync() takes minimum 100ms, we can be faster
		for _, c := range []cache.SharedIndexInformer{
			wfc.wfInformer,
			wfc.wftmplInformer.Informer(),
			wfc.podInformer,
			wfc.cwftmplInformer.Informer(),
			wfc.wfTaskSetInformer.Informer(),
			wfc.artGCTaskInformer.Informer(),
			wfc.taskResultInformer,
		} {
			for !c.HasSynced() {
				time.Sleep(5 * time.Millisecond)
			}
		}

	}
	return cancel, wfc
}

func newControllerWithDefaults() (context.CancelFunc, *WorkflowController) {
	cancel, controller := newController(func(controller *WorkflowController) {
		controller.Config.WorkflowDefaults = &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{HostNetwork: pointer.BoolPtr(true)},
		}
	})
	return cancel, controller
}

func newControllerWithComplexDefaults() (context.CancelFunc, *WorkflowController) {
	cancel, controller := newController(func(controller *WorkflowController) {
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
				HostNetwork:        pointer.BoolPtr(true),
				Entrypoint:         "good_entrypoint",
				ServiceAccountName: "my_service_account",
				TTLStrategy: &wfv1.TTLStrategy{
					SecondsAfterCompletion: pointer.Int32Ptr(10),
					SecondsAfterSuccess:    pointer.Int32Ptr(10),
					SecondsAfterFailure:    pointer.Int32Ptr(10),
				},
			},
		}
	})
	return cancel, controller
}

func newControllerWithDefaultsVolumeClaimTemplate() (context.CancelFunc, *WorkflowController) {
	cancel, controller := newController(func(controller *WorkflowController) {
		controller.Config.WorkflowDefaults = &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{
				VolumeClaimTemplates: []apiv1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "workdir",
					},
					Spec: apiv1.PersistentVolumeClaimSpec{
						AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.ReadWriteOnce},
						Resources: apiv1.ResourceRequirements{
							Requests: apiv1.ResourceList{
								apiv1.ResourceStorage: resource.MustParse("1Mi"),
							},
						},
						StorageClassName: pointer.String("local-path"),
					},
				}},
			},
		}
	})
	return cancel, controller
}

func unmarshalArtifact(yamlStr string) *wfv1.Artifact {
	var artifact wfv1.Artifact
	wfv1.MustUnmarshal([]byte(yamlStr), &artifact)
	return &artifact
}

func expectWorkflow(ctx context.Context, controller *WorkflowController, name string, test func(wf *wfv1.Workflow)) {
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	test(wf)
}

func expectNamespacedWorkflow(ctx context.Context, controller *WorkflowController, namespace, name string, test func(wf *wfv1.Workflow)) {
	wf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	test(wf)
}

func getPod(woc *wfOperationCtx, name string) (*apiv1.Pod, error) {
	return woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func listPods(woc *wfOperationCtx) (*apiv1.PodList, error) {
	return woc.controller.kubeclientset.CoreV1().Pods(woc.wf.Namespace).List(context.Background(), metav1.ListOptions{})
}

type with func(pod *apiv1.Pod)

func withOutputs(v interface{}) with {
	switch x := v.(type) {
	case string:
		return withAnnotation(common.AnnotationKeyOutputs, x)
	default:
		return withOutputs(wfv1.MustMarshallJSON(x))
	}
}
func withProgress(v string) with { return withAnnotation(common.AnnotationKeyProgress, v) }

func withExitCode(v int32) with {
	return func(pod *apiv1.Pod) {
		for _, c := range pod.Spec.Containers {
			pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, apiv1.ContainerStatus{
				Name: c.Name,
				State: apiv1.ContainerState{
					Terminated: &apiv1.ContainerStateTerminated{
						ExitCode: v,
					},
				},
			})
		}
	}
}

func withAnnotation(key, val string) with {
	return func(pod *apiv1.Pod) { pod.Annotations[key] = val }
}

// createRunningPods creates the pods that are marked as running in a given test so that they can be accessed by the
// pod assessor
func createRunningPods(ctx context.Context, woc *wfOperationCtx) {
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	for _, node := range woc.wf.Status.Nodes {
		if node.Type == wfv1.NodeTypePod && node.Phase == wfv1.NodeRunning {
			pod, _ := podcs.Create(ctx, &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: node.ID,
					Annotations: map[string]string{
						"workflows.argoproj.io/node-name": node.Name,
					},
					Labels: map[string]string{
						"workflows.argoproj.io/workflow": woc.wf.Name,
					},
				},
				Status: apiv1.PodStatus{
					Phase: apiv1.PodRunning,
				},
			}, metav1.CreateOptions{})
			_ = woc.controller.podInformer.GetStore().Add(pod)
		}
	}
}

func syncPodsInformer(ctx context.Context, woc *wfOperationCtx, podObjs ...apiv1.Pod) {
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	podObjs = append(podObjs, pods.Items...)
	for _, pod := range podObjs {
		err = woc.controller.podInformer.GetIndexer().Add(&pod)
		if err != nil {
			panic(err)
		}
	}
}

// makePodsPhase acts like a pod controller and simulates the transition of pods transitioning into a specified state
func makePodsPhase(ctx context.Context, woc *wfOperationCtx, phase apiv1.PodPhase, with ...with) {
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase != phase {
			pod.Status.Phase = phase
			if phase == apiv1.PodFailed {
				pod.Status.Message = "Pod failed"
			}
			for _, w := range with {
				w(&pod)
			}
			updatedPod, err := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
			if err != nil {
				panic(err)
			}
			err = woc.controller.podInformer.GetStore().Update(updatedPod)
			if err != nil {
				panic(err)
			}
			if phase == apiv1.PodSucceeded {
				nodeID := woc.nodeID(&pod)
				woc.wf.Status.MarkTaskResultComplete(nodeID)
			}
		}
	}
}

func deletePods(ctx context.Context, woc *wfOperationCtx) {
	for _, obj := range woc.controller.podInformer.GetStore().List() {
		pod := obj.(*apiv1.Pod)
		err := woc.controller.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			panic(err)
		}
		err = woc.controller.podInformer.GetStore().Delete(obj)
		if err != nil {
			panic(err)
		}
	}
}

func TestAddingWorkflowDefaultValueIfValueNotExist(t *testing.T) {
	ans := true
	t.Run("WithoutDefaults", func(t *testing.T) {
		cancel, controller := newController()
		defer cancel()
		workflow := wfv1.MustUnmarshalWorkflow(helloWorldWf)
		err := controller.setWorkflowDefaults(workflow)
		assert.NoError(t, err)
		assert.Equal(t, workflow, wfv1.MustUnmarshalWorkflow(helloWorldWf))
	})
	t.Run("WithDefaults", func(t *testing.T) {
		cancel, controller := newControllerWithDefaults()
		defer cancel()
		defaultWorkflowSpec := wfv1.MustUnmarshalWorkflow(helloWorldWf)
		err := controller.setWorkflowDefaults(defaultWorkflowSpec)
		assert.NoError(t, err)
		assert.Equal(t, defaultWorkflowSpec.Spec.HostNetwork, &ans)
		assert.NotEqual(t, defaultWorkflowSpec, wfv1.MustUnmarshalWorkflow(helloWorldWf))
		assert.Equal(t, *defaultWorkflowSpec.Spec.HostNetwork, true)
	})
}

func TestAddingWorkflowDefaultComplex(t *testing.T) {
	cancel, controller := newControllerWithComplexDefaults()
	defer cancel()
	workflow := wfv1.MustUnmarshalWorkflow(testDefaultWf)
	var ten int32 = 10
	assert.Equal(t, workflow.Spec.Entrypoint, "whalesay")
	assert.Nil(t, workflow.Spec.TTLStrategy)
	assert.Contains(t, workflow.Labels, "foo")
	err := controller.setWorkflowDefaults(workflow)
	assert.NoError(t, err)
	assert.NotEqual(t, workflow, wfv1.MustUnmarshalWorkflow(testDefaultWf))
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
	workflow := wfv1.MustUnmarshalWorkflow(testDefaultWfTTL)
	var ten int32 = 10
	var five int32 = 5
	err := controller.setWorkflowDefaults(workflow)
	assert.NoError(t, err)
	assert.NotEqual(t, workflow, wfv1.MustUnmarshalWorkflow(testDefaultWfTTL))
	assert.Equal(t, workflow.Spec.Entrypoint, "whalesay")
	assert.Equal(t, workflow.Spec.ServiceAccountName, "whalesay")
	assert.Equal(t, *workflow.Spec.TTLStrategy.SecondsAfterCompletion, five)
	assert.Equal(t, *workflow.Spec.TTLStrategy.SecondsAfterFailure, ten)
	assert.NotContains(t, workflow.Labels, "foo")
	assert.Contains(t, workflow.Labels, "label")
	assert.Contains(t, workflow.Annotations, "annotation")
}

func TestAddingWorkflowDefaultVolumeClaimTemplate(t *testing.T) {
	cancel, controller := newControllerWithDefaultsVolumeClaimTemplate()
	defer cancel()
	workflow := wfv1.MustUnmarshalWorkflow(testDefaultWf)
	err := controller.setWorkflowDefaults(workflow)
	assert.NoError(t, err)
	assert.Equal(t, workflow, wfv1.MustUnmarshalWorkflow(testDefaultVolumeClaimTemplateWf))
}

func TestNamespacedController(t *testing.T) {
	kubeClient := fake.Clientset{}
	allowed := false
	kubeClient.AddReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed},
		}, nil
	})

	cancel, controller := newController()
	defer cancel()
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

	cancel, controller := newController()
	defer cancel()
	controller.kubeclientset = kubernetes.Interface(&kubeClient)
	controller.cwftmplInformer = nil
	controller.createClusterWorkflowTemplateInformer(context.TODO())
	assert.NotNil(t, controller.cwftmplInformer)
}

func TestParallelism(t *testing.T) {
	for tt, f := range map[string]func(controller *WorkflowController){
		"Parallelism": func(x *WorkflowController) {
			x.Config.Parallelism = 1
		},
		"NamespaceParallelism": func(x *WorkflowController) {
			x.Config.NamespaceParallelism = 1
		},
	} {
		t.Run(tt, func(t *testing.T) {
			cancel, controller := newController(
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf-0
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
`),
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf-1
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
`),
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf-2
spec:
  shutdown: Terminate
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
`),
				f,
			)
			defer cancel()
			ctx := context.Background()
			assert.True(t, controller.processNextItem(ctx))
			assert.True(t, controller.processNextItem(ctx))
			assert.True(t, controller.processNextItem(ctx))

			expectWorkflow(ctx, controller, "my-wf-0", func(wf *wfv1.Workflow) {
				if assert.NotNil(t, wf) {
					assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)
				}
			})
			expectWorkflow(ctx, controller, "my-wf-1", func(wf *wfv1.Workflow) {
				if assert.NotNil(t, wf) {
					assert.Equal(t, wfv1.WorkflowPending, wf.Status.Phase)
					assert.Equal(t, "Workflow processing has been postponed because too many workflows are already running", wf.Status.Message)
				}
			})
			expectWorkflow(ctx, controller, "my-wf-2", func(wf *wfv1.Workflow) {
				if assert.NotNil(t, wf) {
					assert.Equal(t, wfv1.WorkflowFailed, wf.Status.Phase)
				}
			})
		})
	}
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
  name: workflow-template-hello-world
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
  serviceAccountName: my-sa
  priority: 77
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
  volumes:
  - name: data
    emptyDir: {}
`

func TestCheckAndInitWorkflowTmplRef(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wfTmpl)
	cancel, controller := newController(wf, wftmpl)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)
	err := woc.setExecWorkflow(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, wftmpl.Spec.Templates, woc.execWf.Spec.Templates)
}

const wfWithInvalidMetadataLabelsFrom = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: invalid-labels-from
spec:
  serviceAccountName: my-sa
  entrypoint: test-container
  arguments:
    parameters:
      - name: execution_label
        value: some/special/char
  workflowMetadata:
    labelsFrom:
      execution_label:
        expression: workflow.parameters.execution_label
  templates:
  - name: test-container
    container:
      image: alpine:latest
      command: ["echo", "bye"]
`

const wfWithInvalidMetadataLabels = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: invalid-labels
spec:
  serviceAccountName: my-sa
  entrypoint: test-container
  workflowMetadata:
    labels:
      test: $INVALID
  templates:
  - name: test-container
    container:
      image: alpine:latest
      command: ["echo", "bye"]
`

func TestInvalidWorkflowMetadata(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithInvalidMetadataLabelsFrom)
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)
	err := woc.setExecWorkflow(context.Background())
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "invalid label value")
	}

	wf = wfv1.MustUnmarshalWorkflow(wfWithInvalidMetadataLabels)
	cancel, controller = newController(wf)
	defer cancel()
	woc = newWorkflowOperationCtx(wf, controller)
	err = woc.setExecWorkflow(context.Background())
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "invalid label value")
	}
}

func TestIsArchivable(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	var lblSelector metav1.LabelSelector
	lblSelector.MatchLabels = make(map[string]string)
	lblSelector.MatchLabels["workflows.argoproj.io/archive-strategy"] = "true"

	workflow := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	t.Run("EverythingSelector", func(t *testing.T) {
		controller.archiveLabelSelector = labels.Everything()
		assert.True(t, controller.isArchivable(workflow))
	})
	t.Run("NothingSelector", func(t *testing.T) {
		controller.archiveLabelSelector = labels.Nothing()
		assert.False(t, controller.isArchivable(workflow))
	})
	t.Run("ConfiguredSelector", func(t *testing.T) {
		selector, err := metav1.LabelSelectorAsSelector(&lblSelector)
		assert.NoError(t, err)
		controller.archiveLabelSelector = selector
		assert.False(t, controller.isArchivable(workflow))
		workflow.Labels = make(map[string]string)
		workflow.Labels["workflows.argoproj.io/archive-strategy"] = "true"
		assert.True(t, controller.isArchivable(workflow))
	})
}

func TestReleaseAllWorkflowLocks(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	t.Run("nilObject", func(t *testing.T) {
		controller.releaseAllWorkflowLocks(nil)
	})
	t.Run("unStructuredObject", func(t *testing.T) {
		un := &unstructured.Unstructured{}
		controller.releaseAllWorkflowLocks(un)
	})
	t.Run("otherObject", func(t *testing.T) {
		un := &wfv1.Workflow{}
		controller.releaseAllWorkflowLocks(un)
	})
}

var wfWithSema = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 name: hello-world
 namespace: default
 labels:
   workflows.argoproj.io/completed: false
spec:
 entrypoint: whalesay
 synchronization:
   semaphore:
     configMapKeyRef:
       name: my-config
       key: workflow
 templates:
 - name: whalesay
   container:
     image: docker/whalesay:latest
     command: [cowsay]
     args: ["hello world"]
`

func TestNotifySemaphoreConfigUpdate(t *testing.T) {
	assert := assert.New(t)
	wf := wfv1.MustUnmarshalWorkflow(wfWithSema)
	wf1 := wf.DeepCopy()
	wf1.Name = "one"
	wf2 := wf.DeepCopy()
	wf2.Name = "two"
	wf2.Spec.Synchronization = nil

	cancel, controller := newController(wf, wf1, wf2)
	defer cancel()

	cm := apiv1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      "my-config",
		Namespace: "default",
	}}
	assert.Equal(3, controller.wfQueue.Len())

	// Remove all Wf from Worker queue
	for i := 0; i < 3; i++ {
		key, _ := controller.wfQueue.Get()
		controller.wfQueue.Done(key)
		controller.wfQueue.Forget(key)
	}
	assert.Equal(0, controller.wfQueue.Len())

	controller.notifySemaphoreConfigUpdate(&cm)
	time.Sleep(2 * time.Second)
	assert.Equal(2, controller.wfQueue.Len())
}

func TestParallelismWithInitializeRunningWorkflows(t *testing.T) {
	for tt, f := range map[string]func(controller *WorkflowController){
		"Parallelism": func(x *WorkflowController) {
			x.Config.Parallelism = 1
		},
	} {
		t.Run(tt, func(t *testing.T) {
			cancel, controller := newController(
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf-0
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
`),
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf-1
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
`),
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf-2
  labels:
    workflows.argoproj.io/phase: Running
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
status:
  phase: Running
`),
				f,
			)
			defer cancel()
			ctx := context.Background()

			// process my-wf-0; update status to Pending
			assert.True(t, controller.processNextItem(ctx))
			expectWorkflow(ctx, controller, "my-wf-0", func(wf *wfv1.Workflow) {
				if assert.NotNil(t, wf) {
					assert.Equal(t, wfv1.WorkflowPending, wf.Status.Phase)
					assert.Equal(t, "Workflow processing has been postponed because too many workflows are already running", wf.Status.Message)
				}
			})

			// process my-wf-1; update status to Pending
			assert.True(t, controller.processNextItem(ctx))
			expectWorkflow(ctx, controller, "my-wf-1", func(wf *wfv1.Workflow) {
				if assert.NotNil(t, wf) {
					assert.Equal(t, wfv1.WorkflowPending, wf.Status.Phase)
					assert.Equal(t, "Workflow processing has been postponed because too many workflows are already running", wf.Status.Message)
				}
			})
		})
	}
}

func TestNamespaceParallelismWithInitializeRunningWorkflows(t *testing.T) {
	for tt, f := range map[string]func(controller *WorkflowController){
		"NamespaceParallelism": func(x *WorkflowController) {
			x.Config.NamespaceParallelism = 1
		},
	} {
		t.Run(tt, func(t *testing.T) {
			cancel, controller := newController(
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-ns-0-wf-0
  namespace: ns-0
  creationTimestamp: 2023-06-13T16:39:00Z
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
`),
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-ns-1-wf-0
  namespace: ns-1
  creationTimestamp: 2023-06-13T16:40:00Z
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
`),
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-ns-0-wf-1
  namespace: ns-0
  creationTimestamp: 2023-06-13T16:41:00Z
  labels:
    workflows.argoproj.io/phase: Running
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
status:
  phase: Running
`),
				wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-ns-1-wf-1
  namespace: ns-1
  creationTimestamp: 2023-06-13T16:42:00Z
  labels:
    workflows.argoproj.io/phase: Running
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
status:
  phase: Running
`),
				f,
			)
			defer cancel()
			ctx := context.Background()

			ns0PendingWfTested := false
			ns1PendingWfTested := false
			for {
				assert.True(t, controller.processNextItem(ctx))
				if !ns0PendingWfTested {
					expectNamespacedWorkflow(ctx, controller, "ns-0", "my-ns-0-wf-0", func(wf *wfv1.Workflow) {
						if assert.NotNil(t, wf) {
							if wf.Status.Phase != "" {
								assert.Equal(t, wfv1.WorkflowPending, wf.Status.Phase)
								assert.Equal(t, "Workflow processing has been postponed because too many workflows are already running", wf.Status.Message)
								ns0PendingWfTested = true
							}
						}
					})
				}
				if !ns1PendingWfTested {
					expectNamespacedWorkflow(ctx, controller, "ns-1", "my-ns-1-wf-0", func(wf *wfv1.Workflow) {
						if assert.NotNil(t, wf) {
							if wf.Status.Phase != "" {
								assert.Equal(t, wfv1.WorkflowPending, wf.Status.Phase)
								assert.Equal(t, "Workflow processing has been postponed because too many workflows are already running", wf.Status.Message)
								ns1PendingWfTested = true
							}
						}
					})
				}
				if ns0PendingWfTested && ns1PendingWfTested {
					break
				}
			}
		})
	}
}

func TestPodCleanupRetryIsReset(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: test
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
  `)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	assert.True(t, controller.processNextItem(ctx))

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	woc.operate(ctx)
	assert.True(t, controller.processNextPodCleanupItem(ctx))
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	podCleanupKey := "test/my-wf/labelPodCompleted"
	assert.Equal(t, 0, controller.podCleanupQueue.NumRequeues(podCleanupKey))
}

func TestPodCleanupDeletePendingPodWhenTerminate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: test
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: my-image
  `)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	assert.True(t, controller.processNextItem(ctx))

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	makePodsPhase(ctx, woc, apiv1.PodPending)
	woc.execWf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
	woc.operate(ctx)
	assert.True(t, controller.processNextPodCleanupItem(ctx))
	assert.True(t, controller.processNextPodCleanupItem(ctx))
	assert.True(t, controller.processNextPodCleanupItem(ctx))
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 0)
}

func TestPendingPodWhenTerminate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	wf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
	wf.Status.Phase = wfv1.WorkflowPending

	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	assert.True(t, controller.processNextItem(ctx))

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	for _, node := range woc.wf.Status.Nodes {
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
	}
}

func TestWorkflowReferItselfFromExpression(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(fromExrpessingWf)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	assert.True(t, controller.processNextItem(ctx))

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	woc.operate(ctx)
	assert.True(t, controller.processNextPodCleanupItem(ctx))
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

const podSpecPatchTemplateLevelWf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: wf-
spec:
  entrypoint: wf
  templates:
  - name: wf
    steps:

    - - name: run-task1
        template: run-task
        arguments:
          parameters:
            - name: memreqnum
              value: '25'
            - name: memrequnit
              value: Mi
            - name: message
              value: "hello from run-task1"
  - name: run-task
    inputs:
      parameters:
        - name: memreqnum
        - name: memrequnit
        - name: message
    retryStrategy:
      limit: "2"
      retryPolicy: "Always"
      expression: 'lastRetry.status == "Error" or (lastRetry.status == "Failed" and asInt(lastRetry.exitCode) not in [1,2,127])'
    podSpecPatch: |
      containers:
      - name: main
        resources:
          requests:
            memory: "{{=(sprig.int(retries)+1)*sprig.int(inputs.parameters.memreqnum)}}{{inputs.parameters.memrequnit}}"
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestPodSpecPatchTemplateLevel(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(podSpecPatchTemplateLevelWf)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	assert.True(t, controller.processNextItem(ctx))

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)

	pod := pods.Items[0]
	var mainContainer apiv1.Container
	for _, container := range pod.Spec.Containers {
		if container.Name == "main" {
			mainContainer = container
		}
	}
	require.NotNil(t, mainContainer)
	assert.Equal(t, "25Mi", mainContainer.Resources.Requests.Memory().String())
}
