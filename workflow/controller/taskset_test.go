package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestCreateTaskSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: good
            template: http 
            arguments:
              parameters: [{name: url, value: "https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json"}]
        - - name: bad
            template: http
            continueOn:
              failed: true
            arguments:
              parameters: [{name: url, value: "http://openlibrary.org/people/george08/nofound.json"}]

    - name: http
      inputs:
        parameters:
          - name: url
      http:
       url: "{{inputs.parameters.url}}"

`)
	ctx := logging.TestContext(t.Context())
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal(`apiVersion: argoproj.io/v1alpha1
kind: WorkflowTaskSet
metadata:
  name: http-template-1
  namespace: default
spec:
  tasks:
    http-template-nxvtg-1265710817:
      http:
        url: http://openlibrary.org/people/george08/nofound.json
      inputs:
        parameters:
        - name: url
          value: http://openlibrary.org/people/george08/nofound.json
      name: http
    `, &ts)

	t.Run("CreateTaskSet", func(t *testing.T) {
		cancel, controller := newController(ctx, wf, ts, defaultServiceAccount)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		tslist, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, tslist.Items)
		assert.Len(t, tslist.Items, 1)
		for _, ts := range tslist.Items {
			assert.NotNil(t, ts)
			assert.Equal(t, ts.Name, wf.Name)
			assert.Equal(t, ts.Namespace, wf.Namespace)
			assert.Len(t, ts.Spec.Tasks, 1)
		}
		pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items)
		assert.Len(t, pods.Items, 1)
		for _, pod := range pods.Items {
			assert.NotNil(t, pod)
			assert.True(t, strings.HasSuffix(pod.Name, "-agent"))
		}
	})
	t.Run("CreateTaskSetWithInstanceID", func(t *testing.T) {
		cancel, controller := newController(ctx, wf, ts, defaultServiceAccount)
		defer cancel()
		controller.Config.InstanceID = "testID"
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		tslist, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, tslist.Items)
		assert.Len(t, tslist.Items, 1)
		for _, ts := range tslist.Items {
			assert.NotNil(t, ts)
			assert.Equal(t, ts.Name, wf.Name)
			assert.Equal(t, ts.Namespace, wf.Namespace)
			assert.Len(t, ts.Spec.Tasks, 1)
			assert.Equal(t, "testID", ts.Labels[common.LabelKeyControllerInstanceID], "WorkflowTaskSet should have instanceID label")
		}
		pods, err := woc.controller.kubeclientset.CoreV1().Pods("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items)
		assert.Len(t, pods.Items, 1)
		for _, pod := range pods.Items {
			assert.NotNil(t, pod)
			assert.True(t, strings.HasSuffix(pod.Name, "-agent"))
			assert.Equal(t, "testID", pod.Labels[common.LabelKeyControllerInstanceID])
		}
	})
}

func TestRemoveCompletedTaskSetStatus(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template-1
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: good
            template: http
            arguments:
              parameters: [{name: url, value: "https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json"}]
        - - name: bad
            template: http
            continueOn:
              failed: true
            arguments:
              parameters: [{name: url, value: "http://openlibrary.org/people/george08/nofound.json"}]

    - name: http
      inputs:
        parameters:
          - name: url
      http:
       url: "{{inputs.parameters.url}}"
status:
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  finishedAt: null
  nodes:
    http-template-fqgsf:
      children:
      - http-template-fqgsf-898749974
      displayName: http-template-fqgsf
      finishedAt: null
      id: http-template-fqgsf
      name: http-template-fqgsf
      phase: Running
      startedAt: "2021-07-20T16:05:13Z"
      templateName: main
      templateScope: local/http-template-fqgsf
      type: Steps
    http-template-fqgsf-898749974:
      boundaryID: http-template-fqgsf
      children:
      - http-template-fqgsf-2338098285
      - http-template-fqgsf-3753847819
      displayName: '[0]'
      finishedAt: null
      id: http-template-fqgsf-898749974
      name: http-template-fqgsf[0]
      phase: Running
      startedAt: "2021-07-20T16:05:13Z"
      templateScope: local/http-template-fqgsf
      type: StepGroup
    http-template-fqgsf-2338098285:
      boundaryID: http-template-fqgsf
      displayName: good
      finishedAt: null
      id: http-template-fqgsf-2338098285
      inputs:
        parameters:
        - name: url
          value: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
      name: http-template-fqgsf[0].good
      outputs:
        parameters:
        - name: result
          value: |
            {
              "swagger": "2.0",
              "info": {
                "title": "pkg/apis/workflow/v1alpha1/generated.proto",
                "version": "version not set"
              },
              "consumes": [
                "application/json"
              ],
              "produces": [
                "application/json"
              ],
              "paths": {},
              "definitions": {}
            }
      phase: Succeeded
      startedAt: "2021-07-20T16:05:13Z"
      templateName: http
      templateScope: local/http-template-fqgsf
      type: HTTP
    http-template-fqgsf-3753847819:
      boundaryID: http-template-fqgsf
      displayName: bad
      finishedAt: null
      id: http-template-fqgsf-3753847819
      inputs:
        parameters:
        - name: url
          value: http://openlibrary.org/people/george08/nofound.json
      message: 404 Not Found
      name: http-template-fqgsf[0].bad
      phase: Failed
      startedAt: "2021-07-20T16:05:13Z"
      templateName: http
      templateScope: local/http-template-fqgsf
      type: HTTP
  phase: Running
  progress: 0/0
  startedAt: "2021-07-20T16:05:13Z"
`)
	ctx := logging.TestContext(t.Context())
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal(`apiVersion: argoproj.io/v1alpha1
kind: WorkflowTaskSet
metadata:
  name: http-template-1
  namespace: default
spec:
  tasks:
    http-template-fqgsf-2338098285:
      http:
        url: http://openlibrary.org/people/george08/nofound.json
      inputs:
        parameters:
        - name: url
          value: http://openlibrary.org/people/george08/nofound.json
      name: http
status:
  nodes:
    http-template-fqgsf-2338098285:
      outputs:
        parameters:
        - name: result
          value: |
            {
              "swagger": "2.0",
              "info": {
                "title": "pkg/apis/workflow/v1alpha1/generated.proto",
                "version": "version not set"
              },
              "consumes": [
                "application/json"
              ],
              "produces": [
                "application/json"
              ],
              "paths": {},
              "definitions": {}
            }
      phase: Succeeded
    http-template-fqgsf-3753847819:
      message: 404 Not Found
      phase: Failed

    `, &ts)
	t.Run("RemoveCompletedTaskSetStatus", func(t *testing.T) {
		cancel, controller := newController(ctx, wf, ts)
		defer cancel()
		_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").Create(ctx, &ts, v1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		err = woc.removeCompletedTaskSetStatus(ctx)
		require.NoError(t, err)
		tslist, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").List(ctx, v1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, tslist.Items)
		assert.Len(t, tslist.Items, 1)

		for _, ts := range tslist.Items {
			assert.NotNil(t, ts)
			assert.Equal(t, ts.Name, wf.Name)
			assert.Equal(t, ts.Namespace, wf.Namespace)
			assert.Empty(t, ts.Spec.Tasks)
			assert.Empty(t, ts.Status.Nodes)
		}

	})
}

func TestNonHTTPTemplateScenario(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	t.Run("reconcileTaskSet", func(t *testing.T) {
		woc.operate(ctx)
		err := woc.reconcileTaskSet(ctx)
		require.NoError(t, err)
	})
	t.Run("removeCompletedTaskSetStatus", func(t *testing.T) {
		woc.operate(ctx)
		err := woc.removeCompletedTaskSetStatus(ctx)
		require.NoError(t, err)
	})
}

func TestReconcileTaskSetWithMemoization(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template-1
  namespace: default
spec:
  entrypoint: main
  templates:
    - name: main
      http:
       url: "http://localhost"
      memoize:
        key: cache-demo-1
        maxAge: "10s"
        cache:
          configMap:
            name: cache-demo-1
status:
  nodes:
    http-template-fqgsf-2338098285:
      boundaryID: http-template-fqgsf
      displayName: main
      id: http-template-fqgsf-2338098285
      name: http-template-fqgsf[0].main
      memoizationStatus:
        hit: false
        key: cache-demo-1
        cacheName: cache-demo-1
      outputs:
        parameters:
        - name: result
          value: |
            { demo }
      phase: Succeeded
      templateName: http
      type: HTTP
  phase: Running
`)
	ctx := logging.TestContext(t.Context())
	var ts wfv1.WorkflowTaskSet
	wfv1.MustUnmarshal(`apiVersion: argoproj.io/v1alpha1
kind: WorkflowTaskSet
metadata:
  name: http-template-1
  namespace: default
spec:
  tasks:
    http-template-fqgsf-2338098285:
      http:
        url: http://localhost
      name: http
status:
  nodes:
    http-template-fqgsf-2338098285:
      outputs:
        parameters:
        - name: result
          value: |
            { demo }
      phase: Succeeded
    `, &ts)
	t.Run("MemoizeOnTaskSetSucceeded", func(t *testing.T) {
		cancel, controller := newController(ctx, wf, ts)
		defer cancel()
		_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("default").Create(ctx, &ts, v1.CreateOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		time.Sleep(1 * time.Second)
		err = woc.reconcileTaskSet(ctx)
		require.NoError(t, err)
		memo, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "cache-demo-1", v1.GetOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, memo.Data["cache-demo-1"])
	})
}
