package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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
	return &WorkflowController{
		Config: WorkflowControllerConfig{
			ExecutorImage: "executor:latest",
		},
		kubeclientset: fake.NewSimpleClientset(),
		wfclientset:   fakewfclientset.NewSimpleClientset(),
		completedPods: make(chan string, 512),
	}
}

func marshallBody(b interface{}) io.ReadCloser {
	result, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}
	return ioutil.NopCloser(bytes.NewReader(result))
}

func unmarshalWF(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

// makePodsRunning acts like a pod controller and simulates the transition of pods transitioning into a running state
func makePodsRunning(t *testing.T, kubeclientset kubernetes.Interface, namespace string) {
	podcs := kubeclientset.CoreV1().Pods(namespace)
	pods, err := podcs.List(metav1.ListOptions{})
	assert.Nil(t, err)
	for _, pod := range pods.Items {
		pod.Status.Phase = apiv1.PodRunning
		_, _ = podcs.Update(&pod)
	}
}
