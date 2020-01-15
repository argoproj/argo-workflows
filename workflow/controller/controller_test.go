package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	wfextv "github.com/argoproj/argo/pkg/client/informers/externalversions"
	"github.com/argoproj/argo/workflow/config"
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

func newController() *WorkflowController {
	wfclientset := fakewfclientset.NewSimpleClientset()
	informerFactory := wfextv.NewSharedInformerFactory(wfclientset, 10*time.Minute)
	wftmplInformer := informerFactory.Argoproj().V1alpha1().WorkflowTemplates()
	ctx := context.Background()
	go wftmplInformer.Informer().Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), wftmplInformer.Informer().HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	return &WorkflowController{
		Config: config.WorkflowControllerConfig{
			ExecutorImage: "executor:latest",
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

// makePodsRunning acts like a pod controller and simulates the transition of pods transitioning into a running state
func makePodsRunning(t *testing.T, kubeclientset kubernetes.Interface, namespace string) {
	podcs := kubeclientset.CoreV1().Pods(namespace)
	pods, err := podcs.List(metav1.ListOptions{})
	assert.NoError(t, err)
	for _, pod := range pods.Items {
		pod.Status.Phase = apiv1.PodRunning
		_, _ = podcs.Update(&pod)
	}
}
