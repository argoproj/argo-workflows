package metrics

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

const fakeWf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2019-12-30T19:56:09Z"
  generateName: hello-world-
  generation: 5
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: hello-world-rs795
  namespace: default
  resourceVersion: "1079740"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-rs795
  uid: 7b37bffa-7d03-4f95-b125-35be479a7987
spec:
  arguments: {}
  entrypoint: whalesay
  templates:
  - arguments: {}
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  finishedAt: "2019-12-30T19:56:14Z"
  nodes:
    hello-world-rs795:
      displayName: hello-world-rs795
      finishedAt: "2019-12-30T19:56:13Z"
      id: hello-world-rs795
      name: hello-world-rs795
      phase: Succeeded
      startedAt: "2019-12-30T19:56:09Z"
      templateName: whalesay
      type: Pod
  phase: Succeeded
  startedAt: "2019-12-30T19:56:09Z"
`

const expectedResponse = `# HELP argo_workflow_completion_time Completion time in unix timestamp for a workflow.
# TYPE argo_workflow_completion_time gauge
argo_workflow_completion_time{entrypoint="whalesay",name="hello-world-rs795",namespace="default"} 1.577735774e+09
# HELP argo_workflow_created_time Creation time in unix timestamp for a workflow.
# TYPE argo_workflow_created_time gauge
argo_workflow_created_time{entrypoint="whalesay",name="hello-world-rs795",namespace="default"} 1.577735769e+09
# HELP argo_workflow_info Information about workflow.
# TYPE argo_workflow_info gauge
argo_workflow_info{entrypoint="whalesay",name="hello-world-rs795",namespace="default",service_account_name="",templates="whalesay"} 1
# HELP argo_workflow_start_time Start time in unix timestamp for a workflow.
# TYPE argo_workflow_start_time gauge
argo_workflow_start_time{entrypoint="whalesay",name="hello-world-rs795",namespace="default"} 1.577735769e+09
# HELP argo_workflow_status_phase The workflow current phase.
# TYPE argo_workflow_status_phase gauge
argo_workflow_status_phase{entrypoint="whalesay",name="hello-world-rs795",namespace="default",phase="Error"} 0
argo_workflow_status_phase{entrypoint="whalesay",name="hello-world-rs795",namespace="default",phase="Failed"} 0
argo_workflow_status_phase{entrypoint="whalesay",name="hello-world-rs795",namespace="default",phase="Pending"} 0
argo_workflow_status_phase{entrypoint="whalesay",name="hello-world-rs795",namespace="default",phase="Running"} 0
argo_workflow_status_phase{entrypoint="whalesay",name="hello-world-rs795",namespace="default",phase="Skipped"} 0
argo_workflow_status_phase{entrypoint="whalesay",name="hello-world-rs795",namespace="default",phase="Succeeded"} 1
# HELP argo_workflow_step_completion_time Completion time in unix timestamp for a workflow step.
# TYPE argo_workflow_step_completion_time gauge
argo_workflow_step_completion_time{name="hello-world-rs795",namespace="default",step_name="hello-world-rs795"} 1.577735773e+09
# HELP argo_workflow_step_start_time Start time in unix timestamp for a workflow step.
# TYPE argo_workflow_step_start_time gauge
argo_workflow_step_start_time{name="hello-world-rs795",namespace="default",step_name="hello-world-rs795"} 1.577735769e+09
# HELP argo_workflow_step_status_phase The workflow step current phase.
# TYPE argo_workflow_step_status_phase gauge
argo_workflow_step_status_phase{name="hello-world-rs795",namespace="default",phase="Error",step_name="hello-world-rs795"} 0
argo_workflow_step_status_phase{name="hello-world-rs795",namespace="default",phase="Failed",step_name="hello-world-rs795"} 0
argo_workflow_step_status_phase{name="hello-world-rs795",namespace="default",phase="Pending",step_name="hello-world-rs795"} 0
argo_workflow_step_status_phase{name="hello-world-rs795",namespace="default",phase="Running",step_name="hello-world-rs795"} 0
argo_workflow_step_status_phase{name="hello-world-rs795",namespace="default",phase="Skipped",step_name="hello-world-rs795"} 0
argo_workflow_step_status_phase{name="hello-world-rs795",namespace="default",phase="Succeeded",step_name="hello-world-rs795"} 1
`

func newFakeWorkflow(fakeWf string) *unstructured.Unstructured {
	var wf unstructured.Unstructured
	err := yaml.Unmarshal([]byte(fakeWf), &wf.Object)
	if err != nil {
		panic(err)
	}
	return &wf
}

func newFakeInformer(fakeWf ...string) cache.SharedIndexInformer {
	var fakeWfs []runtime.Object
	for _, name := range fakeWf {
		fakeWfs = append(fakeWfs, newFakeWorkflow(name))
	}
	wfClientSet := fake.NewSimpleDynamicClient(runtime.NewScheme(), fakeWfs...)

	return dynamicinformer.NewDynamicSharedInformerFactory(wfClientSet, 0).ForResource(schema.GroupVersionResource{
		Group:    workflow.Group,
		Version:  "v1alpha1",
		Resource: workflow.WorkflowPlural,
	}).Informer()
}

func TestMetric(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wfInformer := newFakeInformer(fakeWf)
	go wfInformer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), wfInformer.HasSynced) {
		log.Fatal("Timed out waiting for caches to sync")
	}
	registry := NewWorkflowRegistry(wfInformer)
	server := NewServer(ctx, PrometheusConfig{
		Enabled: true,
		Path:    "/metrics",
		Port:    "9090",
	}, registry)

	req, err := http.NewRequest("GET", "/metrics", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	server.Handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Equal(t, expectedResponse, body)
}